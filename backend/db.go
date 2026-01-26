/*
 * db.go - TimescaleDB operacije za EdgeAI sustav
 *
 * Sprema i dohvaća podatke o kvarovima.
 * Koristi pgx connection pool za efikasno upravljanje vezama.
 */
package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB je globalni connection pool
var DB *pgxpool.Pool

// FaultEvent predstavlja jedan detektirani kvar
type FaultEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Abnormal   float64   `json:"abnormal"`   // (0.0-1.0)
	Normal     float64   `json:"normal"`     // referentna vrijednost
	Conclusion string    `json:"conclusion"` // "KVAR"
}

// InitDB spaja se na TimescaleDB i testira konekciju
func InitDB() {
	var err error

	// Kreiranje connection pool s konfiguracijom iz env varijabli
	DB, err = pgxpool.New(context.Background(), cfg.PostgresURL)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	// Testiranje konekcije - ping fail -> baza nije dostupna
	if err = DB.Ping(context.Background()); err != nil {
		log.Fatal("DB ping error:", err)
	}

	log.Println("Database connected successfully")
}

// InsertFaultEvent sprema novi kvar u bazu
func InsertFaultEvent(abnormal, normal float64) {
	// Provjera za negativne vrijednosti
	if abnormal < 0 || normal < 0 {
		log.Println("Invalid values for DB insert")
		return
	}
	// Timeout za spore DB operacije
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Insert s parametrima (SQL injection safe)
	_, err := DB.Exec(ctx,
		`INSERT INTO fault_events (abnormal, normal, conclusion)
		 VALUES ($1, $2, 'KVAR')`,
		abnormal, normal,
	)

	if err != nil {
		log.Println("DB insert error:", err)
	}
}

// GetFaultEvents vraća zadnjih 50 kvarova za dashboard
func GetFaultEvents() []FaultEvent {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query za najnovije kvarove (optimizirano za UI)
	rows, err := DB.Query(ctx,
		`SELECT timestamp, abnormal, normal, conclusion
		 FROM fault_events
		 ORDER BY timestamp DESC
		 LIMIT 50`)

	if err != nil {
		log.Println("DB query error:", err)
		return []FaultEvent{} //Graceful degradation - vraća prazan slice
	}
	defer rows.Close() // Kritično -> oslobodi DB resurse

	// Iteriraj kroz rezultate i popuni slice FaultEvent strukturama
	events := make([]FaultEvent, 0)
	for rows.Next() {
		var e FaultEvent
		if err := rows.Scan(&e.Timestamp, &e.Abnormal, &e.Normal, &e.Conclusion); err != nil {
			log.Println("DB scan error:", err)
			continue
		}
		events = append(events, e)
	}

	return events
}
