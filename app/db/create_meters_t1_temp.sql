CREATE TABLE IF NOT EXISTS meters_t1_temp (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    t1_ea_pos     REAL,
    t1_ea_neg     REAL,
    t1_er_pos     REAL,
    t1_er_neg     REAL,
    t1_es         REAL,
    t1_er         REAL,
    t1_runtime    REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('meters_t1_temp', 'timestamp', if_not_exists => TRUE);
