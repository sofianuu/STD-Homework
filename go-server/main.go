package main

import (
	"context"
	"log"
	"net/http"
)


func main(){
	setupAPI()

	log.Fatal(http.ListenAndServeTLS(":8080","server.crt", "server.key",nil))

}

func setupAPI(){
	ctx := context.Background()

	manager := NewManager(ctx) 

	http.Handle("/", http.FileServer(http.Dir("../vue-chat")))
	http.HandleFunc("/ws", manager.serverWS)
	http.HandleFunc("/login", manager.loginHandler)
}