package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCache_New(t *testing.T) {
	cache := New(100)
	if cache.limit != 100 {
		t.Errorf("ожидали limit=100, получили %d", cache.limit)
	}
	if cache.Cap != 0 {
		t.Error("Cap должен быть 0 при создании")
	}
	if cache.hit != 0 || cache.miss != 0 {
		t.Error("счётчики должны быть 0")
	}
}

func TestCache_SetAndGet_Basic(t *testing.T) {
	cache := New(50)
	ctx := context.TODO()
	go cache.Collector(ctx)

	cache.Set("key1", 42, 10*time.Second)

	val, ok := cache.Get("key1")
	if !ok || val != 42 {
		t.Errorf("Get вернул %d, ok=%v, ожидали 42/true", val, ok)
	}

	if cache.Cap != 1 {
		t.Errorf("Cap должен быть 1, получили %d", cache.Cap)
	}
	if cache.hit != 1 || cache.miss != 0 {
		t.Errorf("счётчики: hit=%d, miss=%d", cache.hit, cache.miss)
	}
}

func TestCache_Set_ExistingKey(t *testing.T) {
	cache := New(50)
	cache.Set("key1", 100, 10*time.Second)

	// пытаемся перезаписать
	cache.Set("key1", 999, 10*time.Second)

	val, ok := cache.Get("key1")
	if !ok || val != 100 { // старое значение!
		t.Errorf("ключ не должен был обновиться! Получили %d", val)
	}
	// проверяем, что в логах было сообщение (но в тесте просто проверим, что значение не поменялось)
}

func TestCache_Get_Miss(t *testing.T) {
	cache := New(50)
	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("не существующий ключ должен возвращать false")
	}
	if cache.miss != 1 {
		t.Errorf("miss должен быть 1, получили %d", cache.miss)
	}
}

func TestCache_Eviction_WhenFull(t *testing.T) {
	cache := New(3) // маленький лимит для теста
	ctx := context.TODO()
	go cache.Collector(ctx)

	// заполняем до лимита
	cache.Set("a", 1, 1*time.Minute) // Used=1
	cache.Set("b", 2, 1*time.Minute) // Used=1
	cache.Set("c", 3, 1*time.Minute) // Used=1

	// делаем "b" популярным
	cache.Get("b") // Used=2
	cache.Get("b") // Used=3

	// добавляем 4-й ключ → должно вытеснить
	cache.Set("d", 4, 1*time.Minute)

	// проверяем, что осталось 3 ключа
	count := 0
	cache.store.Range(func(_, _ interface{}) bool { count++; return true })
	if count != 3 {
		t.Errorf("после eviction должно остаться 3 ключа, осталось %d", count)
	}

	// самый редкий (a) должен был вытесниться, а не b
	_, okA := cache.Get("a")
	if okA {
		t.Error("ключ 'a' (самый редкий) должен был быть вытеснен")
	}
	_, okB := cache.Get("b")
	if !okB {
		t.Error("ключ 'b' (самый частый) должен остаться")
	}
}

func TestCache_Eviction_SameUsed_OldestDeleted(t *testing.T) {
	cache := New(2)
	ctx := context.TODO()
	go cache.Collector(ctx)

	cache.Set("old", 10, 1*time.Minute) // CreatedAt раньше
	time.Sleep(10 * time.Millisecond)   // чтобы CreatedAt отличался
	cache.Set("new", 20, 1*time.Minute) // CreatedAt новее

	// теперь добавляем третий — оба Used=1, должен удалить самый старый
	cache.Set("third", 30, 1*time.Minute)

	_, okOld := cache.Get("old")
	if okOld {
		t.Error("должен был удалиться самый старый ключ 'old'")
	}
	_, okNew := cache.Get("new")
	if !okNew {
		t.Error("новый ключ должен остаться")
	}
}

func TestCache_Expiration_Collector(t *testing.T) {
	cache := New(10)
	ctx := context.TODO()
	go cache.Collector(ctx)

	cache.Set("exp1", 777, 500*time.Millisecond) // TTL = 0.5 сек
	cache.Set("exp2", 888, 2*time.Second)

	time.Sleep(700 * time.Millisecond) // чуть больше 500мс

	// exp1 должен быть удалён коллектором
	_, ok1 := cache.Get("exp1")
	if ok1 {
		t.Error("ключ с TTL 500ms должен был удалиться")
	}

	// exp2 ещё жив
	_, ok2 := cache.Get("exp2")
	if !ok2 {
		t.Error("ключ с TTL 2s ещё не должен был удалиться")
	}

	if cache.Cap != 1 { // только exp2 остался
		t.Errorf("Cap должен быть 1, получили %d", cache.Cap)
	}
}

func TestCache_HitsMisses_AfterExpiration(t *testing.T) {
	cache := New(5)
	ctx := context.TODO()
	go cache.Collector(ctx)

	cache.Set("key", 123, 300*time.Millisecond)

	// первый Get — hit
	cache.Get("key")

	time.Sleep(400 * time.Millisecond)

	// второй Get — miss (уже expired и удалён)
	cache.Get("key")

	if cache.hit != 1 || cache.miss != 1 {
		t.Errorf("ожидали hit=1, miss=1, получили hit=%d, miss=%d", cache.hit, cache.miss)
	}
}

func TestCache_Print(t *testing.T) {
	cache := New(10)
	cache.Set("test1", 100, 10*time.Second)
	cache.Set("test2", 200, 10*time.Second)

	// просто проверяем, что Print не падает
	cache.Print()
}

func TestCache_Concurrent_Access(t *testing.T) {
	cache := New(100)
	ctx := context.TODO()
	go cache.Collector(ctx)

	done := make(chan bool, 100)

	for i := 0; i < 50; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			cache.Set(key, id, 10*time.Second)
			cache.Get(key)
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	if cache.Cap != 50 {
		t.Errorf("после 50 конкурентных Set должно быть 50 записей, получили %d", cache.Cap)
	}
}
