package main

import (
	"flag"
	"log"

	"github.com/jeroenpf/coda-homework-assignment/internal/api"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.Parse()

	cfg := api.Config{
		Port:        port,
		Environment: "development",
		ConsulConfig: api.ConsulConfig{
			Address: "localhost:8500",
		},
	}

	if err := api.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
