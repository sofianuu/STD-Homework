package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"websocket-go/db"
	//"websocket-go/models"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type Manager struct {
	clients ClientList
	sync.RWMutex

	opts RetentionMap

}

func NewManager(ctx context.Context) *Manager {
	return &Manager{
		clients: make(ClientList),
		opts:    NewRetentionMap(ctx, 5*time.Second),
	}
}

func (m *Manager) serverWS(w http.ResponseWriter, r *http.Request) {

	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.opts.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("new connection")

	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)

	m.addClient(client)

	//start client processes
	go client.readMessages()
	go client.writeMessages()

	//conn.Close()
}


func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081") // sau "*" temporar în dev
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ok, _, err := db.LoginUser(req.Username, req.Password)
	if err != nil{
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if ok {

		otp := m.opts.NewOTP()

		type response struct {
			OTP string `json:"otp"`
		}

		resp := response{
			OTP: otp.Key,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)

}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "http://localhost:8080", "https://localhost:8080", "http://localhost:8081", "https://localhost:8081":
		return true
	default:
		return false
	}
}


