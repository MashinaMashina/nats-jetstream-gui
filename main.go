package main

import (
	"embed"
	_ "embed"
	"os"

	"github.com/MashinaMashina/nats-jetstream-gui/pkg/embedutil"
	"github.com/MashinaMashina/nats-jetstream-gui/server"
	"github.com/joho/godotenv"
)

//go:embed public/build/index.html
var indexContent []byte

//go:embed public/build/*
var staticFiles embed.FS

func main() {
	log := server.NewLogger()

	err := godotenv.Load()
	if err != nil {
		log.Error().Err(err).Msg("loading .env file")
	}

	wd, _ := os.Getwd()
	log.Info().Str("dir", wd).Msg("working dir")

	addr := ":8080"
	if val := os.Getenv("NATS_GUI_ADDR"); val != "" {
		addr = val
	}

	natsAddr := "nats://localhost:4222"
	if val := os.Getenv("NATS_GUI_NATS_ADDR"); val != "" {
		natsAddr = val
	}

	api := server.NewAPI(log, natsAddr)
	err = api.Run(addr, indexContent, embedutil.NewPrefixFS("public/build/", staticFiles))
	if err != nil {
		log.Error().Err(err).Msg("running server")
	}
}
