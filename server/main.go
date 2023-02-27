package main

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	log := NewLogger()

	addr := ":8080"
	if val := os.Getenv("NATS_GUI_ADDR"); val != "" {
		addr = val
	}

	server := NewServer(log)
	err := server.Run(addr)
	if err != nil {
		log.Error().Err(err).Msg("running server")
	}
}
