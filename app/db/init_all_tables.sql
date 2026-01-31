-- START: create_measurements.sql --
CREATE TABLE IF NOT EXISTS measurements (
    timestamp     TIMESTAMPTZ       NOT NULL,
    device_id     SMALLINT          NOT NULL,
    f             REAL,
    u1            REAL,
    u2            REAL,
    u3            REAL,
    u12           REAL,
    u23           REAL,
    u31           REAL,
    i1            REAL,
    i2            REAL,
    i3            REAL,
    i_n           REAL,
    p1            REAL,
    p2            REAL,
    p3            REAL,
    q1            REAL,
    q2            REAL,
    q3            REAL,
    s1            REAL,
    s2            REAL,
    s3            REAL,
    pf1           REAL,
    pf2           REAL,
    pf3           REAL,
    p             REAL,
    q             REAL,
    s             REAL,
    pf            REAL,
    PRIMARY KEY (timestamp, device_id)
);

-- Konwersja na hypertable
SELECT create_hypertable('measurements', 'timestamp', if_not_exists => TRUE);


-- START: create_flow.sql --
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


-- START: create_shift_summary.sql --
CREATE TABLE IF NOT EXISTS shift_summary (
    data_utworzenia       TIMESTAMPTZ      NOT NULL,
    start_zmiany          TIMESTAMPTZ      NOT NULL,
    koniec_zmiany         TIMESTAMPTZ      NOT NULL,
    czas_pracy            REAL,
    czas_postoju          REAL,
    czas_przezbrojenia    REAL,
    czas_pomiaru          REAL,
    ilosc_elementow       SMALLINT,
    dostepnosc            REAL,
    wydajnosc             REAL,
    jakosc                REAL,
    oee                   REAL,

    -- dane z 5 analizatorów energii
    analizator_1_t_ea_pos     REAL,
    analizator_1_t_ea_neg     REAL,
    analizator_1_t_er_pos     REAL,
    analizator_1_t_er_neg     REAL,
    analizator_1_t_es         REAL,
    analizator_1_t_er         REAL,

    analizator_2_t_ea_pos     REAL,
    analizator_2_t_ea_neg     REAL,
    analizator_2_t_er_pos     REAL,
    analizator_2_t_er_neg     REAL,
    analizator_2_t_es         REAL,
    analizator_2_t_er         REAL,

    analizator_3_t_ea_pos     REAL,
    analizator_3_t_ea_neg     REAL,
    analizator_3_t_er_pos     REAL,
    analizator_3_t_er_neg     REAL,
    analizator_3_t_es         REAL,
    analizator_3_t_er         REAL,

    analizator_4_t_ea_pos     REAL,
    analizator_4_t_ea_neg     REAL,
    analizator_4_t_er_pos     REAL,
    analizator_4_t_er_neg     REAL,
    analizator_4_t_es         REAL,
    analizator_4_t_er         REAL,

    analizator_5_t_ea_pos     REAL,
    analizator_5_t_ea_neg     REAL,
    analizator_5_t_er_pos     REAL,
    analizator_5_t_er_neg     REAL,
    analizator_5_t_es         REAL,
    analizator_5_t_er         REAL,

    -- dane z przepływomierzy powietrza
    totaliser_1               REAL,
    totaliser_2               REAL,
    totaliser_3               REAL,
    totaliser_4               REAL,
    totaliser_5               REAL,

    PRIMARY KEY (data_utworzenia)
);

-- Konwersja na hypertable
SELECT create_hypertable('shift_summary', 'data_utworzenia', if_not_exists => TRUE);

-- Indeksy pomocnicze
CREATE INDEX IF NOT EXISTS idx_shift_summary_start_zmiany ON shift_summary(start_zmiany);
CREATE INDEX IF NOT EXISTS idx_shift_summary_koniec_zmiany ON shift_summary(koniec_zmiany);


-- START: create_oee_temp.sql --
CREATE TABLE IF NOT EXISTS oee_temp (
    timestamp            TIMESTAMPTZ      NOT NULL,
    predkosc_obrotnica   REAL,
    czas_pracy           REAL,
    czas_postoju         REAL,
    czas_pomiaru         REAL,
    czas_przezbrojenia   REAL,
    status_maszyny       BOOLEAN,
    ilosc_elementow      SMALLINT,
    dlugosc_calc         REAL,
    szerokosc_calc       REAL,
    wysokosc_calc        REAL,
    dostepnosc           REAL,
    wydajnosc            REAL,
    jakosc               REAL,
    cykl                 REAL,
    oee                  REAL,
    PRIMARY KEY (timestamp)
);

-- Konwersja na hypertable
SELECT create_hypertable('oee_temp', 'timestamp', if_not_exists => TRUE);


-- START: create_meters_total_temp.sql --
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


-- START: create_meters_t1_temp.sql --
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


-- START: create_meters_t2_temp.sql --
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


-- START: create_meters_t3_temp.sql --
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


-- START: create_meters_t4_temp.sql --
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


