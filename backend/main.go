/*
 *	main.go - glavni kod
 *
 *	Upravljanje servisima: DB, MQTT, WebSocket, HTTP
 *  Implementacija graceful shutdown
 *	Upravljanje životnim ciklusom resursa
 */
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Globalni - dijeljeni resurs za MQTT operacije
var mqttClient mqtt.Client

func main() {
	InitDB()         // Inicijalizacija konekcije na TimescaleDB
	defer DB.Close() // Osiguranje da se baza zatvori pri gašenju programa

	log.Println("Database initialized, WebSocket service ready")

	// Pokretanje MQTT konekcije i background worker-a
	mqttClient = StartMQTT()
	defer mqttClient.Disconnect(250)

	// Pokretanje goroutine koja automatski šalje status klijentima
	StartStatusBroadcaster()

	// Pokretanje HTTP servera u gorutini (non-blocking)
	router := SetupRouter() // Postavljanje Gin routera sa endpointima
	go func() {
		log.Printf("HTTP server running on %s\n", cfg.ServerAddr)
		if err := router.Run(cfg.ServerAddr); err != nil {
			log.Fatal("HTTP server failed:", err)
		}
	}()

	// Graceful shutdown - zatvara faultChan (zaustavlja workere), ostalo ide preko defer-a (MQTT, DB).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Initiating graceful shutdown...")
	// Kritično - prvo zatvori faultChan da workeri znaju da staju
	close(faultChan)
	log.Println("Shutdown complete")
}
