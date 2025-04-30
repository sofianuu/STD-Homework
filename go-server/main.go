package main

import ( 
	"net/http"
	"log"
)


func main(){
	setupAPI()

	log.Fatal(http.ListenAndServe(":8080",nil))

}

func setupAPI(){
	manager := NewManager() 

	http.Handle("/", http.FileServer(http.Dir("../vue-chat")))
	http.HandleFunc("/ws", manager.serverWS)
}