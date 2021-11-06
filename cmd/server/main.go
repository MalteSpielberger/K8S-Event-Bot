package main

import "k8s-event-bot/internal/server"

func main() {
	srv := server.NewServer()

	srv.Start()
}