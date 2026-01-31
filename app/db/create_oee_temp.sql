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
