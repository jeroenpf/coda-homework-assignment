package main

import (
	"coda-homework-assignment/internal/api"
	"flag"
	"log"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.Parse()

	cfg := api.Config{
		Port: port,
	}

	if err := api.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
