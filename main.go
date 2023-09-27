package main

import (
	"embed"
	_ "embed"
	"os"

	"github.com/MashinaMashina/nats-jetstream-gui/pkg/embedutil"
	"github.com/MashinaMashina/nats-jetstream-gui/server"
	_ "github.com/joho/godotenv/autoload"
)

//go:embed public/build/index.html
var indexContent []byte

//go:embed public/build/*
var staticFiles embed.FS

func main() {
	log := server.NewLogger()

	wd, _ := os.Getwd()
	log.Info().Str("dir", wd).Msg("working dir")

	addr := ":8080"
	if val := os.Getenv("NATS_GUI_ADDR"); val != "" {
		addr = val
	}

	api := server.NewAPI(log)
	err := api.Run(addr, indexContent, embedutil.NewPrefixFS("public/build/", staticFiles))
	if err != nil {
		log.Error().Err(err).Msg("running server")
	}
}
