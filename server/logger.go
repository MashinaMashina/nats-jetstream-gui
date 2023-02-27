package server

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger() zerolog.Logger {
	output := os.Getenv("NATS_GUI_LOGGER_OUTPUT")
	level := "trace"
	if val := os.Getenv("NATS_GUI_LOGGER_LEVEL"); val != "" {
		level = val
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimestampFieldName = "_t"
	zerolog.LevelFieldName = "_l"
	zerolog.MessageFieldName = "_m"
	zerolog.ErrorFieldName = "_e"

	var w io.Writer
	if output == "" {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	} else {
		w = &lumberjack.Logger{
			Filename:   output,
			MaxSize:    500, // megabytes
			MaxAge:     28,  //days
			MaxBackups: 3,
		}
	}

	currentLevel := zerolog.InfoLevel
	if l, err := zerolog.ParseLevel(strings.ToLower(level)); level != "" && err == nil {
		currentLevel = l
	}

	log.Logger = zerolog.New(w).With().Timestamp().Logger().Level(currentLevel)
	zerolog.SetGlobalLevel(currentLevel)

	return log.Logger
}
