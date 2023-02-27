package server

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewClient(conn *websocket.Conn) *WsClient {
	return &WsClient{
		conn: conn,
	}
}

func (w *WsClient) Write(msg []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return 0, err
	}

	return len(msg), nil
}
