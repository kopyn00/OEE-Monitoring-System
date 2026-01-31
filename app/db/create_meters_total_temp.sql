CREATE TABLE IF NOT EXISTS meters_total_temp (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    ea_pos_total  REAL,
    ea_neg_total  REAL,
    er_pos_total  REAL,
    er_neg_total  REAL,
    es_total      REAL,
    er_total      REAL,
    ea_pos        REAL,
    ea_neg        REAL,
    er_pos        REAL,
    er_neg        REAL,
    es            REAL,
    er            REAL,
    e_runtime     REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('meters_total_temp', 'timestamp', if_not_exists => TRUE);
