-- migrations/001_init.sql
CREATE TABLE IF NOT EXISTS logins (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    app_version VARCHAR(50) NOT NULL,
    device_type VARCHAR(50) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    locale VARCHAR(10) NOT NULL,
    original_timestamp BIGINT NOT NULL,
    parsed_timestamp TIMESTAMP NOT NULL,
    processed_at TIMESTAMP NOT NULL
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_user_logins ON logins (user_id, parsed_timestamp);
CREATE INDEX IF NOT EXISTS idx_device_logins ON logins (device_id, parsed_timestamp);
CREATE INDEX IF NOT EXISTS idx_ip_logins ON logins (ip, parsed_timestamp);