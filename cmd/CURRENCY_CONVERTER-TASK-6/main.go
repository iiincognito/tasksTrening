package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Current struct {
	EURrub float64
	USDrub float64

	convertHistory map[string]int
}

func NewCurrent() *Current {
	return &Current{
		convertHistory: make(map[string]int),
	}
}

func (c *Current) RubToUSD(rub float64) {
	c.convertHistory["RubToUSD"]++
	fmt.Println(rub / c.USDrub)
}
func (c *Current) RubToEUR(rub float64) {
	c.convertHistory["RubToEUR"]++
	fmt.Println(rub / c.EURrub)
}
func (c *Current) UsdToRUB(usd float64) {
	c.convertHistory["UsdToRUB"]++
	fmt.Println(usd * c.USDrub)
}
func (c *Current) EurToRUB(eur float64) {
	c.convertHistory["EurToRUB"]++
	fmt.Println(eur * c.EURrub)
}

type CBRRates struct {
	Rates map[string]float64 `json:"rates"`
}

func NewCBR() *CBRRates {
	return &CBRRates{
		Rates: make(map[string]float64),
	}
}

func (c *Current) CurrentRates() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		resp, err := http.Get("https://www.cbr-xml-daily.ru/latest.js")
		if err != nil {
			fmt.Println("exit")
			return
		}
		defer resp.Body.Close()

		var tmp = NewCBR()

		err = json.NewDecoder(resp.Body).Decode(&tmp)
		if err != nil {
			fmt.Println(err)
			return
		}
		if usdRate, ok := tmp.Rates["USD"]; ok && usdRate != 0 {
			fmt.Printf("USD: %f\n", usdRate)
			c.USDrub = 1 / usdRate
		}
		if eurRate, ok := tmp.Rates["EUR"]; ok && eurRate != 0 {
			fmt.Printf("EUR: %f\n", eurRate)
			c.EURrub = 1 / eurRate
		}
		<-ticker.C
	}
}

func main() {
	tmp := NewCurrent()
	go tmp.CurrentRates()
	time.Sleep(1 * time.Second)
	fmt.Println(tmp.USDrub)
	tmp.RubToUSD(1000)
}
