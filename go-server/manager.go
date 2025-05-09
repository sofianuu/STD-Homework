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
	username := m.opts.GetUsername(otp)

	client := NewClient(conn, m, username)

	m.addClient(client)

	messages, err := db.GetMessages()
	if err != nil {
		log.Printf("Error getting last messages: %v", err)
	}else {
		historyJSON, err := json.Marshal(map[string]interface{}{
			"type": "history",
			"messages": messages,
		})
		if err != nil {
			log.Printf("Error marshaling history: %v", err)
		} else {
			client.egress <- historyJSON
		}
	}

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

		m.opts.SetUsername(otp.Key, req.Username )

		type response struct {
			OTP string `json:"otp"`
			Username string `json:"username"`
		}

		resp := response{
			OTP: otp.Key,
			Username: req.Username,
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

func (m *Manager) registerHandler(w http.ResponseWriter, r *http.Request){
	
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081") // sau "*" temporar în dev
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	type userRegisterRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var req userRegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" || req.Email == "" {
		http.Error(w, "Username, password and email are required", http.StatusBadRequest)
		return 
	}

	user, err := db.RegisterUser(req.Username, req.Password, req.Email) 
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		http.Error(w, err.Error(),http.StatusBadRequest)
		return
	}

	type response struct {
		Succes bool `json:"success"`
		Message string `json:"message"`
		User struct {
			ID string `json:"id"`
			Username string `json:"username"`
			Email string `json:"email"`
		} `json:"user"`
	}

	resp := response{
		Succes: true,
		Message: "User registered succesfully",
		User: struct{ID string "json:\"id\""; Username string "json:\"username\""; Email string "json:\"email\""}{
			ID: user.ID.Hex(),
			Username: user.Username,
			Email: user.Email,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)

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


