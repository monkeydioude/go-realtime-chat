package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	mux := http.NewServeMux()
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	mux.Handle("/chat/", http.StripPrefix("/chat/", fs))

	// WebSocket route for /chat/ws
	mux.HandleFunc("/chat/ws", handleConnections)
	// Start listening for incoming chat messages
	go handleMessages()

	server := &http.Server{
		Addr:           ":8000",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// Start the server on localhost port 8000 and log any errors
	cacheDir := "certs"
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      "monkeydioude@gmail.com",
		HostPolicy: autocert.HostWhitelist("4thehoard.com"), //Your domain here
		Cache:      autocert.DirCache(cacheDir),             //Folder for storing certificates
	}
	server.TLSConfig = certManager.TLSConfig()
	server.Addr = ":https"

	// serve = func() error {
	// 	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	// 	return server.ListenAndServeTLS("", "")
	// }
	log.Println("http server started on :8000")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
