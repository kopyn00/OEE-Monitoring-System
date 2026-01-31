CREATE TABLE IF NOT EXISTS flow_data (
    timestamp      TIMESTAMPTZ       NOT NULL,
    device_id      SMALLINT          NOT NULL,
    flow           REAL,
    pressure       REAL,
    temperature    REAL,
    totaliser      REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('flow_data', 'timestamp', if_not_exists => TRUE);
