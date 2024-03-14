package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/MashinaMashina/nats-jetstream-gui/pkg/middleware"
	"github.com/rs/zerolog"
)

type Response struct {
	Error    *string `json:"error"`
	Response any     `json:"response"`
}

type API struct {
	ws   *Websockets
	nats *Nats
	log  zerolog.Logger
}

func NewAPI(log zerolog.Logger) *API {
	natsAddr := "nats://localhost:4222"
	if val := os.Getenv("NATS_GUI_NATS_ADDR"); val != "" {
		natsAddr = val
	}

	return &API{
		ws:   NewWebsockets(log),
		nats: NewNats(log, natsAddr),
		log:  log,
	}
}

func (s *API) Run(addr string, indexContent []byte, staticFiles fs.FS) error {
	go s.TranslateStatistics()

	http.HandleFunc("/api/read/", middleware.AnyCORS(s.form(s.ReadStreamMessage)))
	http.HandleFunc("/api/send/", middleware.AnyCORS(s.form(s.SendStreamMessage)))
	http.HandleFunc("/api/stream_info/", middleware.AnyCORS(s.form(s.StreamInfo)))
	http.HandleFunc("/api/streams/", middleware.AnyCORS(s.form(s.ActiveStreams)))
	http.HandleFunc("/api/delete_stream/", middleware.AnyCORS(s.form(s.DeleteStream)))
	http.HandleFunc("/api/consumers/", middleware.AnyCORS(s.form(s.ActiveConsumers)))
	http.HandleFunc("/api/delete_consumer/", middleware.AnyCORS(s.form(s.DeleteConsumer)))
	http.HandleFunc("/ws/", middleware.AnyCORS(s.ws.OpenConnection))

	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexContent)
	})

	s.log.Info().Str("addr", addr).Msg("starting server")

	return http.ListenAndServe(addr, nil)
}

func (s *API) ActiveStreams(w http.ResponseWriter, r *http.Request) {
	s.response(w, nil, s.nats.ActiveStreams())
}

func (s *API) StreamInfo(w http.ResponseWriter, r *http.Request) {
	stream := r.FormValue("stream")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}

	info, err := s.nats.StreamInfo(stream)

	s.response(w, err, info)
}

func (s *API) ReadStreamMessage(w http.ResponseWriter, r *http.Request) {
	subject := r.FormValue("subject")

	if subject == "" {
		s.response(w, fmt.Errorf("subject in query not specified"), nil)
		return
	}

	ack := false
	if strAck := r.URL.Query().Get("ack"); strAck == "1" {
		ack = true
	}

	msg, err := s.nats.ReadMessage(subject, ack)
	s.response(w, err, msg)
}

func (s *API) SendStreamMessage(w http.ResponseWriter, r *http.Request) {
	subject := r.FormValue("subject")

	if subject == "" {
		s.response(w, fmt.Errorf("subject in query not specified"), nil)
		return
	}

	err := s.nats.SendMessage(StreamMessage{
		Subject: subject,
		Data:    r.FormValue("data"),
	})
	s.response(w, err, nil)
}

func (s *API) TranslateStatistics() {
	stats := s.nats.Statistics(context.Background())

	for stat := range stats {
		s.ws.SendAll(WsMessage{
			Type:    MessageTypeStatistic,
			Message: stat,
		})
	}
}

func (s *API) ActiveConsumers(w http.ResponseWriter, r *http.Request) {
	stream := r.FormValue("stream")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}

	info := s.nats.ActiveConsumers(stream)

	s.response(w, nil, info)
}

func (s *API) DeleteConsumer(w http.ResponseWriter, r *http.Request) {
	stream := r.FormValue("stream")
	consumer := r.FormValue("consumer")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}
	if consumer == "" {
		s.response(w, fmt.Errorf("consumer in query not specified"), nil)
		return
	}

	err := s.nats.DeleteConsumer(stream, consumer)

	s.response(w, err, nil)
}

func (s *API) DeleteStream(w http.ResponseWriter, r *http.Request) {
	stream := r.FormValue("stream")

	if stream == "" {
		s.response(w, fmt.Errorf("stream in query not specified"), nil)
		return
	}

	err := s.nats.DeleteStream(stream)

	s.response(w, err, nil)
}

func (s *API) response(w http.ResponseWriter, err error, resp any) {
	var msg *string
	if err != nil {
		errMsg := err.Error()
		msg = &errMsg
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Error:    msg,
		Response: resp,
	})
}

func (s *API) form(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if err := r.ParseMultipartForm(1e7); err != nil {
				s.response(w, fmt.Errorf("parsing form data: %w", err), nil)
				return
			}
		}

		next(w, r)
	}
}
