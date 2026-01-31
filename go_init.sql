-- Wymagane dla create_hypertable:
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 1) flow_data
CREATE TABLE IF NOT EXISTS public.flow_data (
    timestamp   TIMESTAMPTZ NOT NULL,
    device_id   SMALLINT    NOT NULL,
    flow        REAL,
    pressure    REAL,
    temperature REAL,
    totaliser   REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.flow_data','timestamp', if_not_exists => true);

-- 2) measurements
CREATE TABLE IF NOT EXISTS public.measurements (
    timestamp TIMESTAMPTZ NOT NULL,
    device_id SMALLINT    NOT NULL,
    f   REAL, u1  REAL, u2  REAL, u3  REAL,
    u12 REAL, u23 REAL, u31 REAL,
    i1  REAL, i2  REAL, i3  REAL, i_n REAL,
    p1  REAL, p2  REAL, p3  REAL,
    q1  REAL, q2  REAL, q3  REAL,
    s1  REAL, s2  REAL, s3  REAL,
    pf1 REAL, pf2 REAL, pf3 REAL,
    p   REAL, q   REAL, s   REAL, pf  REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.measurements','timestamp', if_not_exists => true);

-- 3) meters_t1_temp
CREATE TABLE IF NOT EXISTS public.meters_t1_temp (
    timestamp  TIMESTAMPTZ NOT NULL,
    device_id  SMALLINT    NOT NULL,
    t1_ea_pos  REAL,
    t1_ea_neg  REAL,
    t1_er_pos  REAL,
    t1_er_neg  REAL,
    t1_es      REAL,
    t1_er      REAL,
    t1_runtime REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.meters_t1_temp','timestamp', if_not_exists => true);

-- 4) meters_t2_temp
CREATE TABLE IF NOT EXISTS public.meters_t2_temp (
    timestamp  TIMESTAMPTZ NOT NULL,
    device_id  SMALLINT    NOT NULL,
    t2_ea_pos  REAL,
    t2_ea_neg  REAL,
    t2_er_pos  REAL,
    t2_er_neg  REAL,
    t2_es      REAL,
    t2_er      REAL,
    t2_runtime REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.meters_t2_temp','timestamp', if_not_exists => true);

-- 5) meters_t3_temp
CREATE TABLE IF NOT EXISTS public.meters_t3_temp (
    timestamp  TIMESTAMPTZ NOT NULL,
    device_id  SMALLINT    NOT NULL,
    t3_ea_pos  REAL,
    t3_ea_neg  REAL,
    t3_er_pos  REAL,
    t3_er_neg  REAL,
    t3_es      REAL,
    t3_er      REAL,
    t3_runtime REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.meters_t3_temp','timestamp', if_not_exists => true);

-- 6) meters_t4_temp
CREATE TABLE IF NOT EXISTS public.meters_t4_temp (
    timestamp  TIMESTAMPTZ NOT NULL,
    device_id  SMALLINT    NOT NULL,
    t4_ea_pos  REAL,
    t4_ea_neg  REAL,
    t4_er_pos  REAL,
    t4_er_neg  REAL,
    t4_es      REAL,
    t4_er      REAL,
    t4_runtime REAL,
    PRIMARY KEY (timestamp, device_id)
);
SELECT create_hypertable('public.meters_t4_temp','timestamp', if_not_exists => true);

-- 7) meters_total_temp
CREATE TABLE IF NOT EXISTS public.meters_total_temp (
    timestamp     TIMESTAMPTZ NOT NULL,
    device_id     SMALLINT    NOT NULL,
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
SELECT create_hypertable('public.meters_total_temp','timestamp', if_not_exists => true);

-- 8) oee_temp  (PK tylko timestamp)
CREATE TABLE IF NOT EXISTS public.oee_temp (
    timestamp               TIMESTAMPTZ NOT NULL DEFAULT now(),
    predkosc_obrotnica      REAL,
    czas_pracy              REAL,
    czas_postoju            REAL,
    czas_pomiaru            REAL,
    czas_przezbrojenia      REAL,
    status_maszyny          BOOLEAN,
    ilosc_elementow         SMALLINT,
    dlugosc_calc            REAL,
    szerokosc_calc          REAL,
    wysokosc_calc           REAL,
    dostepnosc              REAL,
    wydajnosc               REAL,
    jakosc                  REAL,
    cykl                    REAL,
    oee                     REAL,
    czas_przezbrojenia_temp REAL,
    status_pracy            BOOLEAN,
    w_na_szt           REAL,
    l_na_szt           REAL,
    PRIMARY KEY (timestamp)
);
SELECT create_hypertable('public.oee_temp','timestamp', if_not_exists => true);

-- 9) shift_summary (PK data_utworzenia)
CREATE TABLE IF NOT EXISTS public.shift_summary (
    data_utworzenia    TIMESTAMPTZ NOT NULL DEFAULT now(),
    start_zmiany       TIMESTAMPTZ NOT NULL,
    koniec_zmiany      TIMESTAMPTZ NOT NULL,
    czas_pracy         REAL,
    czas_postoju       REAL,
    czas_przezbrojenia REAL,
    czas_pomiaru       REAL,
    ilosc_elementow    SMALLINT,
    dostepnosc         REAL,
    wydajnosc          REAL,
    jakosc             REAL,
    oee                REAL,

    analizator_1_ea_pos REAL,
    analizator_1_ea_neg REAL,
    analizator_1_er_pos REAL,
    analizator_1_er_neg REAL,
    analizator_1_es     REAL,
    analizator_1_er     REAL,
    analizator_2_ea_pos REAL,
    analizator_2_ea_neg REAL,
    analizator_2_er_pos REAL,
    analizator_2_er_neg REAL,
    analizator_2_es     REAL,
    analizator_2_er     REAL,
    analizator_3_ea_pos REAL,
    analizator_3_ea_neg REAL,
    analizator_3_er_pos REAL,
    analizator_3_er_neg REAL,
    analizator_3_es     REAL,
    analizator_3_er     REAL,
    totaliser_1 REAL,
    totaliser_2 REAL,
    totaliser_3 REAL,
    totaliser_4 REAL,
    totaliser_5 REAL,
    # OEE / jednostkowe
    w_na_szt           REAL,
    l_na_szt           REAL,
    cykl0              REAL,
    cykl1              REAL,
    cykl2              REAL,
    cykl3              REAL,
    PRIMARY KEY (data_utworzenia)
);
SELECT create_hypertable('public.shift_summary','data_utworzenia', if_not_exists => true);