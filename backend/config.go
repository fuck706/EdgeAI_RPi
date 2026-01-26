/*
 * config.go - Konfiguracija za EdgeAI sustav prediktivnog održavanja
 *
 * Učitava postavke iz env varijabli za MQTT (nRF5340), TimescaleDB i dashboard.
 * Lozinke se mogu postaviti kao bcrypt hash (DASH_PASS_HASH) ili plaintext.
 */

package main

import (
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Config drži sve postavke aplikacije
type Config struct {
	// MQTT - postavke
	MQTTBroker   string `json:"mqtt_broker,omitempty"`
	MQTTTopic    string `json:"mqtt_topic,omitempty"`
	MQTTClient   string `json:"mqtt_client,omitempty"`
	MQTTUsername string `json:"mqtt_username,omitempty"`
	MQTTPassword string `json:"mqtt_password,omitempty"`

	// Baza podataka
	PostgresURL string `json:"postgres_url,omitempty"` // TimescaleDB

	//HTTP server
	ServerAddr string `json:"server_addr,omitempty"`

	// Dashboard login
	DashboardUser     string `json:"dashboard_user,omitempty"`
	DashboardPass     string `json:"-"`
	DashboardPassHash string `json:"-"`
}

// Globalna konfiguracija
var cfg = initConfig()

// initConfig učitava konfiguraciju iz env varijabli
func initConfig() Config {
	c := Config{
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTTopic:    getEnv("MQTT_TOPIC", "edgeai/fault"),
		MQTTClient:   getEnv("MQTT_CLIENT", "edgeai-rpi"),
		MQTTUsername: getEnv("MQTT_USERNAME", "username"),
		MQTTPassword: getEnv("MQTT_PASSWORD", "pass"),

		PostgresURL: getEnv("POSTGRES_URL",
			"postgres://edgeai:pass@localhost:5432/edgeai?sslmode=disable"),

		ServerAddr: getEnv("SERVER_ADDR", "0.0.0.0:8080"),

		DashboardUser: getEnv("DASH_USER", "admin"),
		DashboardPass: getEnv("DASH_PASS", "admin12345678"),
	}

	// HASH LOZINKE - prvo DASH_PASS_HASH, pa plaintext fallback
	hashFromEnv := os.Getenv("DASH_PASS_HASH")
	if hashFromEnv != "" {
		c.DashboardPassHash = hashFromEnv
		log.Println("CONFIG: Using pre-hashed DASH_PASS_HASH from environment")
	} else {
		h, err := bcrypt.GenerateFromPassword([]byte(c.DashboardPass), bcrypt.DefaultCost)
		if err != nil {
			panic("FATAL: Failed to hash dashboard password: " + err.Error())
		}
		c.DashboardPassHash = string(h)
		log.Println("CONFIG: Generated bcrypt hash from DASH_PASS (use DASH_PASS_HASH in production)")
	}
	log.Printf("CONFIG: Dashboard auth configured for user '%s' (bcrypt hash: %v)",
		c.DashboardUser, c.DashboardPassHash != "")

	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}
