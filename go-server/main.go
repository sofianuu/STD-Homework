package main

import (
	"context"
	"log"
	"net/http"
	"websocket-go/db"
)

func main() {
	setupAPI()
	db.Connect()

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func setupAPI() {
	ctx := context.Background()

	manager := NewManager(ctx)

	http.Handle("/", http.FileServer(http.Dir("../vue-chat")))
	http.HandleFunc("/ws", manager.serverWS)
	http.HandleFunc("/login", manager.loginHandler)
	http.HandleFunc("/register", manager.registerHandler)
}
