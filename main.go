package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var rootCmd = &cobra.Command{
	Use:   "kubernetes-grafana-updater",
	Short: "update grafana datasources and dashboards",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		l, err := newLogger(currentLogLevel.Level)
		if err != nil {
			panic(err)
		}
		logger = l
	},
}

// Hackery for setting log level
type logLevel struct {
	zapcore.Level
}

var logger *zap.Logger

var currentLogLevel = &logLevel{Level: zapcore.InfoLevel}

func (ll *logLevel) Type() string {
	return "logLevel"
}

func main() {
	rootCmd.PersistentFlags().VarP(currentLogLevel, "log-level", "l", "log level")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func errorLog(format string, a ...interface{}) {
	log.Fatalf(format, a)
}

func newLogger(lvl zapcore.Level) (*zap.Logger, error) {
	config := zap.Config{
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		Encoding:          "json",
		ErrorOutputPaths:  []string{"stdout"},
		Level:             zap.NewAtomicLevel(),
		OutputPaths:       []string{"stdout"},
	}
	config.Level.SetLevel(lvl)

	l, err := config.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logger")
	}
	return l, nil
}
