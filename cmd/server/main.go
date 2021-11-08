package main

import "k8sbot/internal/server"

func main() {
	srv := server.NewServer()

	if err := srv.Start(); err != nil {
		panic(err)
	}
}
