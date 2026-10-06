CREATE TABLE daily_entries (
                               id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                               user_id BIGINT NOT NULL,
                               text    VARCHAR(500) NOT NULL,
                               date    DATE NOT NULL,
                               rating  INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 10),
                               CONSTRAINT fk_daily_entries_user
                                   FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                               CONSTRAINT unique_user_date UNIQUE (user_id, date)
);