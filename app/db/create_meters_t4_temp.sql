CREATE TABLE IF NOT EXISTS meters_t4_temp (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    t4_ea_pos     REAL,
    t4_ea_neg     REAL,
    t4_er_pos     REAL,
    t4_er_neg     REAL,
    t4_es         REAL,
    t4_er         REAL,
    t4_runtime    REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('meters_t4_temp', 'timestamp', if_not_exists => TRUE);
