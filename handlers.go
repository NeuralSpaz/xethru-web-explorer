package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// func indexHandler(w http.ResponseWriter, r *http.Request) {
// 	// os.OpenFile("./index.html", flag, perm)
// 	file, _ := Asset("www/index.html")
// 	w.Header().Set("Content-Type", "text/html")
// 	w.Write(file)
// }
// func websocketReconnectHandler(w http.ResponseWriter, r *http.Request) {
// 	file, _ := Asset("www/js/reconnecting-websocket.min.js")
// 	w.Header().Set("Content-Type", "text/javascript")
// 	w.Write(file)
// }

var respirationconnectionsMutex sync.Mutex
var respirationconnections map[*websocket.Conn]bool

func respirationwsHandler(w http.ResponseWriter, r *http.Request) {
	// Taken from gorilla's website
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	log.Println("Succesfully upgraded connection")
	respirationconnectionsMutex.Lock()
	respirationconnections[conn] = true
	respirationconnectionsMutex.Unlock()
	for {
		// Blocks until a message is read
		_, msg, err := conn.ReadMessage()
		if err != nil {
			respirationconnectionsMutex.Lock()
			// log.Printf("Disconnecting %v because %v\n", conn, err)
			delete(respirationconnections, conn)
			respirationconnectionsMutex.Unlock()
			conn.Close()
			return
		}
		log.Println(msg)
	}
}

func sendrespiration(msg []byte) {
	respirationconnectionsMutex.Lock()
	for conn := range respirationconnections {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			delete(respirationconnections, conn)
			conn.Close()
		}
	}
	respirationconnectionsMutex.Unlock()
}

var baseBandconnectionsMutex sync.Mutex
var baseBandconnections map[*websocket.Conn]bool

func baseBandwsHandler(w http.ResponseWriter, r *http.Request) {
	// Taken from gorilla's website
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	log.Println("Succesfully upgraded connection")
	baseBandconnectionsMutex.Lock()
	baseBandconnections[conn] = true
	baseBandconnectionsMutex.Unlock()

	for {
		// Blocks until a message is read
		_, msg, err := conn.ReadMessage()
		if err != nil {
			baseBandconnectionsMutex.Lock()
			// log.Printf("Disconnecting %v because %v\n", conn, err)
			delete(baseBandconnections, conn)
			baseBandconnectionsMutex.Unlock()
			conn.Close()
			return
		}
		log.Println(msg)
	}
}

func sendBaseBand(msg []byte) {
	baseBandconnectionsMutex.Lock()
	for conn := range baseBandconnections {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			delete(baseBandconnections, conn)
			conn.Close()
		}
	}
	baseBandconnectionsMutex.Unlock()
}

var sleepconnectionsMutex sync.Mutex
var sleepconnections map[*websocket.Conn]bool

func sleepwsHandler(w http.ResponseWriter, r *http.Request) {
	// Taken from gorilla's website
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	log.Println("Succesfully upgraded connection")
	sleepconnectionsMutex.Lock()
	sleepconnections[conn] = true
	sleepconnectionsMutex.Unlock()
	for {
		// Blocks until a message is read
		_, msg, err := conn.ReadMessage()
		if err != nil {
			sleepconnectionsMutex.Lock()
			// log.Printf("Disconnecting %v because %v\n", conn, err)
			delete(sleepconnections, conn)
			sleepconnectionsMutex.Unlock()
			conn.Close()
			return
		}
		log.Println(msg)
	}
}

func sendsleep(msg []byte) {
	sleepconnectionsMutex.Lock()
	for conn := range sleepconnections {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			delete(sleepconnections, conn)
			conn.Close()
		}
	}
	sleepconnectionsMutex.Unlock()
}
