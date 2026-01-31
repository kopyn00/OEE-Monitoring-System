CREATE TABLE IF NOT EXISTS meters_t2_temp (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    t2_ea_pos     REAL,
    t2_ea_neg     REAL,
    t2_er_pos     REAL,
    t2_er_neg     REAL,
    t2_es         REAL,
    t2_er         REAL,
    t2_runtime    REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('meters_t2_temp', 'timestamp', if_not_exists => TRUE);
