package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Clients map[*websocket.Conn]bool

	Broadcast chan []byte

	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn

	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
		Clients:    make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			log.Println("Cliente conectado. Total:", len(h.Clients))

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Println("Cliente desconectado. Total:", len(h.Clients))

		case message := <-h.Broadcast:
			h.mu.Lock()
			for client := range h.Clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Println("Error enviando WS:", err)
					client.Close()
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) SendUpdate(data interface{}) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Println("Error codificando JSON para WS:", err)
		return
	}
	h.Broadcast <- jsonBytes
}
