package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "www.bitmex.com", "http service address")

// Cache ...
type Cache struct {
	sync.RWMutex
	pair              string
	diffAmount        float64
	defaultExpiration time.Duration
	items             map[string]data
}

type data struct {
	cost float64   `json:"pastPrice"`
	time time.Time `json:"timestamp"`
}

type bitmexResponse struct {
	lastPrice string `json:"lastPrice"`
	timestamp string `json:"timestamp"`
}

var (
	btcCache *Cache
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	btcCache = newCache("BTCUSD", 4, 14)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/realtime"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			btcCache.addTicker(message)
		}
	}()

	if err := c.WriteMessage(websocket.TextMessage, []byte("{\"op\": \"subscribe\", \"args\": [\"instrument:XBTUSD\"]}")); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func newCache(pair string, timeout int64, diffAmount float64) *Cache {
	i := make(map[string]data)
	exp := time.Duration(time.Second) * time.Duration(timeout)
	return &Cache{
		items:             i,
		pair:              pair,
		diffAmount:        diffAmount,
		defaultExpiration: exp,
	}
}

func (c *Cache) addTicker(ticker []byte) {
	c.Lock()
	defer c.Unlock()

	newData := &bitmexResponse{}

	err := json.Unmarshal(ticker, newData)
	if err != nil {
		log.Printf("error when marshalled ticker: %s", ticker, err)
	}

	log.Printf(newData.lastPrice)

	analyze(ticker)

	// u := fmt.Sprintf("%s", uuid.NewV4())
	// cost := strconv.ParseFloat(newData.cost, 64)
	// bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	// cost := (ask + bid) / 2
	// c.items[u] = data{cost: toFixed(cost, 5), time: time.Now().Add(c.defaultExpiration)}
	// c.verify(u)
}

func analyze(message []byte) {
	log.Printf("recv: %s", message)
}
