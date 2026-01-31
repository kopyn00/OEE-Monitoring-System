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
    analizator_1_ea_pos     REAL,
    analizator_1_ea_neg     REAL,
    analizator_1_er_pos     REAL,
    analizator_1_er_neg     REAL,
    analizator_1_es         REAL,
    analizator_1_er         REAL,
    analizator_2_ea_pos     REAL,
    analizator_2_ea_neg     REAL,
    analizator_2_er_pos     REAL,
    analizator_2_er_neg     REAL,
    analizator_2_es         REAL,
    analizator_2_er         REAL,
    analizator_3_ea_pos     REAL,
    analizator_3_ea_neg     REAL,
    analizator_3_er_pos     REAL,
    analizator_3_er_neg     REAL,
    analizator_3_es         REAL,
    analizator_3_er         REAL,
    analizator_4_ea_pos     REAL,
    analizator_4_ea_neg     REAL,
    analizator_4_er_pos     REAL,
    analizator_4_er_neg     REAL,
    analizator_4_es         REAL,
    analizator_4_er         REAL,
    analizator_5_ea_pos     REAL,
    analizator_5_ea_neg     REAL,
    analizator_5_er_pos     REAL,
    analizator_5_er_neg     REAL,
    analizator_5_es         REAL,
    analizator_5_er         REAL,

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
