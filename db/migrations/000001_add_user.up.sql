CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR NOT NULL UNIQUE,
    password VARCHAR(60) NOT NULL
);