/*
 * websocket.go - Real-time komunikacija za frontend (dashboard)
 *
 * Održava WebSocket veze s klijentima i broadcast-a kvarove i status automatski
 * Koristi heartbeat (ping/pong) za održavanje veza
 */
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Upgrader pretvara HTTP konekciju u WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Mapa aktivnih WebSocket konekcija
var (
	wsClients = make(map[*websocket.Conn]bool)
	wsMutex   sync.Mutex
)

// cleanup zatvara konekciju i uklanja iz mape
func cleanupClient(conn *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if _, ok := wsClients[conn]; ok {
		conn.Close()
		delete(wsClients, conn)
		log.Println("WebSocket client disconnected")
	}
}

// broadcastMessage šalje poruku SVIM klijentima
func broadcastMessage(msgType string, data interface{}) {
	// Pakiranje poruke u JSON
	msg, err := json.Marshal(map[string]interface{}{
		"type": msgType,
		"data": data,
	})
	if err != nil {
		log.Println("Failed to marshal WS message:", err)
		return
	}
	wsMutex.Lock()
	defer wsMutex.Unlock()
	for conn := range wsClients {
		// Mrtva veza - cleanup
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(wsClients, conn)
		}
	}
}

// WebSocketHandler prima nove klijente
func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	// Dodavanje klijenta u mapu
	wsMutex.Lock()
	wsClients[conn] = true
	wsMutex.Unlock()
	log.Println("WebSocket client connected")
	defer cleanupClient(conn)

	//ping/pong (keep-alive)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	//ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Gorilla WebSocket ZAHTIJEVA da se čitaju poruke, inače browser prekida konekciju.
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	//Ping loop
	for range pingTicker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

// BroadcastFault šalje novi kvar na dashboard
func BroadcastFault(event FaultEvent) {
	broadcastMessage("NEW_FAULT", event)
}

// BroadcastStatus šalje MQTT/DB status
func BroadcastStatus() {
	status := map[string]string{
		"mqtt": "NOT CONNECTED",
		"db":   "ERROR",
	}
	// Provjeravanje baze
	if err := DB.Ping(context.Background()); err == nil {
		status["db"] = "OK"
	}
	// Provjeravanje MQTT-a
	if mqttClient != nil && mqttClient.IsConnected() {
		status["mqtt"] = "CONNECTED"
	}
	broadcastMessage("STATUS", status)
}

// StartStatusBroadcaster šalje status svake 2 sekunde
func StartStatusBroadcaster() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			BroadcastStatus()
		}
	}()
}
