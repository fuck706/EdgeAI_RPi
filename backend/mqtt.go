/*
 * mqtt.go - MQTT klijent za komunikaciju s nRF5340DK
 *
 * Prima podatke od senzora (vibracije) i šalje ih na obradu kroz faultChan buffer.
 */

package main

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// faultChan buffer za spikeove s MQTT-a.
// Ako se napuni, poruke se jednostavno preskoče.
var faultChan = make(chan FaultPayload, 100)

// FaultPayload - struktura podataka s MCU-a
type FaultPayload struct {
	Abnormal   float64 `json:"abnormal"`   // Vrijednost abnormalnosti (0.0-1.0)
	Normal     float64 `json:"normal"`     // referentna vrijednost
	Conclusion string  `json:"conclusion"` // "OK" (normalan rad) ili "KVAR"
}

// faultWorker obrađuje poruke iz kanala
func faultWorker() {
	for payload := range faultChan {
		// Filter -> samo kvarove šalje dalje
		if payload.Conclusion != "KVAR" {
			continue
		}

		// Spremanje u bazu
		InsertFaultEvent(payload.Abnormal, payload.Normal)
		event := FaultEvent{
			Timestamp:  time.Now(),
			Abnormal:   payload.Abnormal,
			Normal:     payload.Normal,
			Conclusion: payload.Conclusion,
		}
		// Push na dashboard
		BroadcastFault(event)
	}
}

// StartMQTT spajanje na broker i primanje podatka s nRF5340
func StartMQTT() mqtt.Client {
	// Pokretanje worker gorutine u pozadini
	go faultWorker()

	// MQTT konfiguracija
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID(cfg.MQTTClient)

	// autentifikacija
	if cfg.MQTTUsername != "" && cfg.MQTTPassword != "" {
		opts.SetUsername(cfg.MQTTUsername)
		opts.SetPassword(cfg.MQTTPassword)
	}
	client := mqtt.NewClient(opts)

	// Pokušava se spojiti na broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT connection error:", token.Error())
	}

	log.Println("Connected to MQTT broker")

	// Pretplata na topic za primanje poruka s MCU-a
	// QoS = 0
	client.Subscribe(cfg.MQTTTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var payload FaultPayload

		// Izdvojanje JSON poruke
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			log.Println("MQTT JSON error:", err)
			return
		}

		// Asinkrono prosljeđivanje u processing pipeline
		// Select pattern sprječavanje blokiranja ako je buffer pun
		select {
		case faultChan <- payload:
			// Normalan flow - poruka je dodana u buffer
		default:
			log.Println("Warning: faultChan buffer full, skipped MQTT message")
		}
	})

	return client
}
