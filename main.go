package main

import (
	"embed"
	_ "embed"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"nats-jetstream-gui/pkg/embedutil"
	"nats-jetstream-gui/server"
)

//go:embed public/build/index.html
var indexContent []byte

//go:embed public/build/*
var staticFiles embed.FS

func main() {
	log := server.NewLogger()

	addr := ":8080"
	if val := os.Getenv("NATS_GUI_ADDR"); val != "" {
		addr = val
	}

	serv := server.NewServer(log)
	err := serv.Run(addr, indexContent, embedutil.NewPrefixFS("public/build/", staticFiles))
	if err != nil {
		log.Error().Err(err).Msg("running server")
	}
}
