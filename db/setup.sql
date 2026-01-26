/*
 * schema.sql - TimescaleDB shema
 * 
 * Hypertables za senzorske podatke (vibracije) i detektirane kvarove.
 * Automatsko particioniranje po vremenu za brže upite i lakše upravljanje.
 */

-- TimescaleDB ekstenzija
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- sirovi podaci sa senzora vibracija (GY-521 na nRF5340DK)
CREATE TABLE IF NOT EXISTS sensor_data (
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),  
    x DOUBLE PRECISION NOT NULL,                   -- X os akcelerometra
    y DOUBLE PRECISION NOT NULL,                   -- Y os 
    z DOUBLE PRECISION NOT NULL,                   -- Z os 
    abnormal DOUBLE PRECISION,                     -- TinyML rezultat (0.0-1.0)
    normal DOUBLE PRECISION,                       -- Referentna vrijednost
    PRIMARY KEY (timestamp)
);

-- Hypertable za senzorske podatke (particije po danu)
SELECT create_hypertable(
    'sensor_data', 
    'timestamp',
    if_not_exists => TRUE,
    chunk_time_interval => interval '1 day'
);

-- Detektirani kvarovi (popunjava se iz Go backend-a)
CREATE TABLE IF NOT EXISTS fault_events (
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    abnormal DOUBLE PRECISION NOT NULL, 
    normal DOUBLE PRECISION NOT NULL,             
    conclusion TEXT NOT NULL                      
);

SELECT create_hypertable(
    'fault_events',
    'timestamp',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_sensor_abnormal 
ON sensor_data(abnormal DESC, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_faults_conclusion 
ON fault_events(conclusion, timestamp DESC);