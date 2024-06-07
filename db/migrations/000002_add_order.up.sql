CREATE TYPE status_type AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE IF NOT EXISTS orders (
    number VARCHAR PRIMARY KEY,
    status status_type NOT NULL DEFAULT 'NEW',
    accrual NUMERIC,
    user_id uuid references users(id),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);