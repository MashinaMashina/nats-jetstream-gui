package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type MessageType string

const MessageTypeStatistic MessageType = "statistic"

type Message struct {
	Type    MessageType `json:"type"`
	Message any         `json:"message"`
}

type Websockets struct {
	clients map[*WsClient]struct{}
	mu      sync.Mutex
	log     zerolog.Logger
}

func NewWebsockets(log zerolog.Logger) *Websockets {
	return &Websockets{
		clients: map[*WsClient]struct{}{},
		log:     log,
	}
}

func (ws *Websockets) OpenConnection(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()

	client := NewClient(connection)

	ws.mu.Lock()
	ws.clients[client] = struct{}{} // Сохраняем соединение, используя его как ключ
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		delete(ws.clients, client) // Удаляем соединение
		ws.mu.Unlock()
	}()

	for {
		mt, message, err := connection.ReadMessage()

		if err != nil || mt == websocket.CloseMessage {
			break // Выходим из цикла, если клиент пытается закрыть соединение или связь прервана
		}

		go ws.onMessage(client, message)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Пропускаем любой запрос
	},
}

func (ws *Websockets) onMessage(client *WsClient, message []byte) {
	var msg Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		ws.log.Error().Bytes("msg", message).Err(err).Msg("cannot unmarshal message")
		return
	}

	ws.Send(client, msg)
}

func (ws *Websockets) SendAll(message Message) {
	for client := range ws.clients {
		ws.Send(client, message)
	}
}

func (ws *Websockets) Send(writer io.Writer, message Message) {
	err := json.NewEncoder(writer).Encode(message)
	if err != nil {
		ws.log.Error().Err(err).Msg("sending message to websocket client")
	}
}
