package main

import "C"
import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type Item struct {
	Value     int
	Used      int
	TTL       time.Duration
	CreatedAt time.Time
}

func (item *Item) IsExpired() bool {
	return time.Now().After(item.CreatedAt.Add(item.TTL))
}

type Cache struct {
	mu    *sync.RWMutex
	Cap   int
	limit int
	store sync.Map
	hit   int
	miss  int
}

func New(limit int) *Cache {
	return &Cache{
		mu:    &sync.RWMutex{},
		limit: limit,
		store: sync.Map{},
	}
}

func (c *Cache) Print() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.store.Range(func(k, v interface{}) bool {
		key := k.(string)
		item := v.(*Item)
		fmt.Println("key =", key, ",", "val =", item.Value, ",", "used =", item.Used)
		return true
	})
}

func (c *Cache) Get(key string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	res, ok := c.store.Load(key)
	if !ok {
		c.miss++
		return 0, false
	}
	item, ok := res.(*Item)
	if !ok {
		return 0, false
	}

	if item.IsExpired() {
		c.store.Delete(key)
		c.Cap--
		c.miss++
		return 0, false
	}
	item.Used++
	c.hit++
	return item.Value, true
}

func (c *Cache) Set(key string, value int, ttl time.Duration) {

	_, ok := c.store.Load(key)
	if ok {
		log.Println("key is not exist")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Cap == c.limit {
		minn := math.MaxInt
		c.store.Range(func(k, v interface{}) bool {
			item := v.(*Item)
			fmt.Println(item.Used)
			if item.Used < minn {
				minn = item.Used
			}
			return true
		})
		tmp := make(map[string]*Item, 10)
		c.store.Range(func(k, v interface{}) bool {
			keey := k.(string)
			item := v.(*Item)
			if item.Used <= minn {
				tmp[keey] = item
			}
			return true
		})

		if len(tmp) == 1 {
			for k := range tmp {
				c.store.Delete(k)
				c.store.Store(key, &Item{
					Value:     value,
					Used:      1,
					TTL:       ttl,
					CreatedAt: time.Now(),
				})
				return
			}
		}

		timeTmp := time.Now()
		count := ""

		for i, v := range tmp {
			if v.CreatedAt.Before(timeTmp) {
				timeTmp = v.CreatedAt
				count = i
			}
		}
		c.store.Delete(count)
		c.store.Store(key, &Item{
			Value:     value,
			Used:      1,
			TTL:       ttl,
			CreatedAt: time.Now(),
		})
		return
	}

	c.store.Store(key, &Item{
		Value:     value,
		Used:      1,
		TTL:       ttl,
		CreatedAt: time.Now(),
	})
	c.Cap++
	return
}

func (c *Cache) Collector(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	tmp := []string{}
	for {
		select {
		case <-ticker.C:
			fmt.Println("запуск коллектора")
			c.mu.RLock()
			c.store.Range(func(k, v interface{}) bool {
				key := k.(string)
				value := v.(*Item)
				if time.Since(value.CreatedAt) > value.TTL {
					tmp = append(tmp, key)
				}
				return true
			})
			c.mu.RUnlock()

			if len(tmp) == 0 {
				continue
			}
			for _, v := range tmp {
				c.store.Delete(v)
				c.Cap--
			}
			tmp = []string{}

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	cache := New(50)
	ctx := context.TODO()
	go cache.Collector(ctx)
	cache.Set("1", 1, 2*time.Second)
	cache.Set("2", 12, 2*time.Second)
	cache.Print()
	fmt.Println(cache.Cap, cache.limit, cache.miss, cache.hit)
	time.Sleep(6 * time.Second)
	cache.Print()
}
