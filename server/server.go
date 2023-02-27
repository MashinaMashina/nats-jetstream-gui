package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"

	"github.com/rs/zerolog"
)

type Response struct {
	Error    *string `json:"error"`
	Response any     `json:"response"`
}

type Server struct {
	ws   *Websockets
	nats *Nats
	log  zerolog.Logger
}

func NewServer(log zerolog.Logger) *Server {
	natsAddr := "nats://localhost:4222"
	if val := os.Getenv("NATS_GUI_NATS_ADDR"); val != "" {
		natsAddr = val
	}
	natsMonitorAddr := "http://localhost:8222"
	if val := os.Getenv("NATS_GUI_MONITOR_ADDR"); val != "" {
		natsMonitorAddr = val
	}

	return &Server{
		ws:   NewWebsockets(log),
		nats: NewNats(log, natsAddr, natsMonitorAddr),
		log:  log,
	}
}

func (s *Server) Run(addr string) error {
	go s.TranslateStatistics()

	http.HandleFunc("/read/", s.ReadStreamMessage)
	http.HandleFunc("/stream_info/", s.StreamInfo)
	http.HandleFunc("/streams/", s.GetStreams)
	http.HandleFunc("/ws/", s.ws.OpenConnection)

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./public/build/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/build/index.html")
	})

	s.log.Info().Str("addr", addr).Msg("starting server")

	return http.ListenAndServe(addr, nil)
}

func (s *Server) GetStreams(w http.ResponseWriter, r *http.Request) {
	s.response(w, nil, s.nats.GetActiveStreams())
}

func (s *Server) StreamInfo(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}

	info, err := s.nats.StreamInfo(stream)

	s.response(w, err, info)
}

func (s *Server) ReadStreamMessage(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}

	s.response(w, s.nats.ReadOneMessage(stream), nil)
}

func (s *Server) TranslateStatistics() {
	stats := s.nats.Statistics(context.Background())

	for stat := range stats {
		stat.Messages /= int64(rand.Intn(4) + 1)
		stat.Bytes /= int64(rand.Intn(400) + 1)

		s.ws.SendAll(Message{
			Type:    MessageTypeStatistic,
			Message: stat,
		})
	}
}

func (s *Server) response(w io.Writer, err error, resp any) {
	var msg *string
	if err != nil {
		errMsg := err.Error()
		msg = &errMsg
	}

	json.NewEncoder(w).Encode(Response{
		Error:    msg,
		Response: resp,
	})
}
