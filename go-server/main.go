package main

import (
	"context"
	"log"
	"net/http"

)


func main(){
	setupAPI()
 //db.InitMongo()

	log.Fatal(http.ListenAndServe(":8080",nil))

}

func setupAPI(){
	ctx := context.Background()

	manager := NewManager(ctx) 

	http.Handle("/", http.FileServer(http.Dir("../vue-chat")))
	http.HandleFunc("/ws", manager.serverWS)
	http.HandleFunc("/login", manager.loginHandler)
}

