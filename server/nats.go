package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Nats struct {
	conn        *nats.Conn
	js          nats.JetStreamContext
	monitorAddr string
	httpClient  HTTPClient
	log         zerolog.Logger
	active      bool
}

func NewNats(log zerolog.Logger, natsAddr, monitorAddr string) *Nats {
	nonActive := &Nats{active: false}
	natsconn, err := nats.Connect(natsAddr,
		nats.Timeout(time.Second*10),
		nats.RetryOnFailedConnect(false),
	)

	if err != nil {
		log.Error().Err(fmt.Errorf("connecting to nats: %w", err)).Send()
		return nonActive
	}

	jetstream, err := natsconn.JetStream()
	if err != nil {
		log.Error().Err(fmt.Errorf("open nats jetstream: %w", err)).Send()
		return nonActive
	}

	return &Nats{
		conn:        natsconn,
		js:          jetstream,
		monitorAddr: monitorAddr,
		httpClient:  http.DefaultClient,
		log:         log,
	}
}

func (n *Nats) ReadOneMessage(stream string) error {
	if err := n.checkActive(); err != nil {
		return err
	}

	subscribe, err := n.js.PullSubscribe(stream, "nats-gui")
	if err != nil {
		return fmt.Errorf("subscribe to stream: %w", err)
	}
	msgs, err := subscribe.Fetch(1)
	if err != nil {
		return fmt.Errorf("fetch message: %w", err)
	}

	fmt.Println(msgs)

	subscribe.Unsubscribe()
	subscribe.Drain()

	return nil
}

type StreamInfo struct {
	Name     string `json:"name"`
	Messages uint64 `json:"messages"`
}

func (n *Nats) GetActiveStreams() []StreamInfo {
	if err := n.checkActive(); err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	streamsChan := n.js.Streams()
	streams := make([]StreamInfo, 0, 8)

cycle:
	for {
		select {
		case info, ok := <-streamsChan:
			if !ok {
				break cycle
			}

			streams = append(streams, StreamInfo{
				Name:     info.Config.Name,
				Messages: info.State.Msgs,
			})
		case <-ctx.Done():
			break cycle
		}
	}

	return streams
}

func (n *Nats) StreamInfo(stream string) (*nats.StreamInfo, error) {
	if err := n.checkActive(); err != nil {
		return nil, err
	}

	return n.js.StreamInfo(stream)
}

type JetStreamStat struct {
	Time      int64 `json:"time"`
	Streams   int64 `json:"streams"`
	Consumers int64 `json:"consumers"`
	Messages  int64 `json:"messages"`
	Bytes     int64 `json:"bytes"`
}

func (n *Nats) Statistics(ctx context.Context) <-chan JetStreamStat {
	c := make(chan JetStreamStat)

	if err := n.checkActive(); err != nil {
		close(c)
		return c
	}

	go func() {
		defer close(c)

		var err error
		var stat JetStreamStat
		for {
			if err = n.monitorRequestJson("jsz", &stat); err != nil {
				n.log.Error().Err(err).Msg("getting monitor statistics")
				return
			}

			stat.Time = time.Now().Unix()

			c <- stat

			select {
			case <-ctx.Done():
				return

			case <-time.After(time.Second):
				continue
			}
		}
	}()

	return c
}

func (n *Nats) monitorRequestJson(path string, res any) error {
	bytes, err := n.monitorRequest(path)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}

	return json.Unmarshal(bytes, res)
}

func (n *Nats) monitorRequest(path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	url := fmt.Sprintf("%s/%s", n.monitorAddr, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	response, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response code (%d)", response.StatusCode)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return bytes, nil
}

func (n *Nats) checkActive() error {
	if !n.active {
		return fmt.Errorf("connect not active")
	}

	return nil
}
