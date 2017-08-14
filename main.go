package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kubernetes-grafana-updater",
	Short: "update grafana datasources and dashboards",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func errorLog(format string, a ...interface{}) {
	log.Fatalf(format, a)
}
