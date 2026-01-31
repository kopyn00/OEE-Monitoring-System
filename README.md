# OEE Monitoring System (Go)

A real-time production monitoring system for calculating **Overall Equipment Effectiveness (OEE)** using data collected from **MQTT (IO-Link sensors)** and **REST APIs (energy analyzers)**.
The system stores time-series data in **PostgreSQL / TimescaleDB** and is fully containerized using **Docker Compose**.

The project is designed for industrial environments and supports continuous data acquisition, OEE calculation, shift-based aggregation, and visualization via Grafana.

---

## Features

* Reads machine and sensor data from **MQTT brokers** (IO-Link masters).
* Retrieves electrical and energy measurements via **REST API**.
* Calculates OEE indicators:

  * `availability`
  * `performance`
  * `quality`
  * `cycle`
  * overall `OEE`
* Aggregates and stores data in:

  * PostgreSQL / TimescaleDB
  * Runtime JSON files (diagnostics and fallback)
* Automatic **3-shift detection and shift summary**.
* Handles **CET / CEST** time zones (Polish local logic).
* Fully dockerized with configuration via `.env`.

---

## Project Structure

```
app/                # Go application source code
├── communication/  # MQTT and REST communication
├── config/         # Application configuration
├── core/           # OEE logic, shift scheduler, data processing
├── db/             # SQL schema and initialization scripts
├── utils/          # Helpers (JSON, conversions, logging)
├── main.go
├── go.mod
├── go.sum
└── Dockerfile

deploy/             # Docker Compose and deployment files
docs/               # Technical documentation
grafana_queries/    # SQL queries for Grafana dashboards
```

---

## Installation (Docker Compose)

### 1. Clone the repository

```bash
git clone <repository-url>
cd OEE-Monitoring-System
```

### 2. Configure environment variables

Create a `.env` file in the repository root (see `env.example`):

```env
# Grafana
GRAFANA_USER=admin
GRAFANA_PASSWORD=change_me

# Database
DB_HOST=timescaledb_go
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=change_me
DB_NAME=oee_monitoring

# MQTT
MQTT_BROKER=192.168.1.100
MQTT_PORT=1883
MQTT_USER=mqtt_user
MQTT_PASSWORD=change_me

# Energy analyzers (REST)
ANALYZER_IP01=192.168.1.201
ANALYZER_IP02=192.168.1.202
ANALYZER_IP03=192.168.1.203
ANALYZER_IP04=192.168.1.204
ANALYZER_IP05=192.168.1.205

# Timezone
TZ=Europe/Warsaw
```

---

### 3. Start the system

```bash
docker compose -f deploy/docker-compose.yml up --build -d
```

---

### 4. Access services

* **Grafana**: [http://localhost:3000](http://localhost:3000)
  (credentials from `.env`)
* **PostgreSQL / TimescaleDB**: port `5432`

---

## Requirements (non-Docker)

* Go 1.18+
* PostgreSQL with TimescaleDB extension
* MQTT broker (e.g. Mosquitto, IO-Link master)
* REST-enabled energy analyzers
  (mock data is supported for development)

---

## Database Overview

The system uses **TimescaleDB** to store time-series production data, including:

* OEE calculations and shift summaries
* Electrical measurements and energy counters
* Flow and auxiliary sensor data

SQL initialization scripts are located in `app/db/`.

---

## Grafana

Grafana dashboards visualize:

* OEE indicators (availability, performance, quality)
* Energy consumption
* Flow and temperature trends

Example SQL queries are stored in `grafana_queries/`.

---

## Author

Developed by **kopyn00** in Go for monitoring industrial production systems using real-time IoT data and OEE metrics.

> ⚠️ Ensure Docker, Docker Compose, and environment variables are properly configured before starting the system.
