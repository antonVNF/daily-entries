CREATE TABLE users (
                       id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                       email      VARCHAR(200) NOT NULL UNIQUE,
                       name       VARCHAR(200) NOT NULL DEFAULT '',
                       created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);