CREATE TABLE IF NOT EXISTS withdrawals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    number VARCHAR NOT NULL,
    sum NUMERIC,
    user_id uuid references users(id),
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);