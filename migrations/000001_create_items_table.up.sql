CREATE TABLE IF NOT EXISTS items (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    model text NOT NULL,
    supplier integer NOT NULL,
    price numeric(64) NOT NULL,
    currency integer NOT NULL,
    image_file text,
    notes text,
    tags text[],
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    archived boolean NOT NULL DEFAULT false
);