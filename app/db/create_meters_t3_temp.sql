CREATE TABLE IF NOT EXISTS meters_t3_temp (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    t3_ea_pos     REAL,
    t3_ea_neg     REAL,
    t3_er_pos     REAL,
    t3_er_neg     REAL,
    t3_es         REAL,
    t3_er         REAL,
    t3_runtime    REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('meters_t3_temp', 'timestamp', if_not_exists => TRUE);
