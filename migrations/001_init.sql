-- migrations/001_init.sql

\echo 'Starting database initialization...';

-- Conditionally create the loginapp user if it doesn't exist
DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE rolname = 'loginapp'
   ) THEN
      CREATE USER loginapp WITH PASSWORD 'securepassword';
   END IF;
END
$do$;

\echo 'Ensured loginapp user exists...';

-- Grant necessary permissions (safe to run multiple times)
GRANT ALL PRIVILEGES ON DATABASE logindb TO loginapp;

\echo 'Granted database permissions...';

-- Connect as the application user to create tables
\connect logindb loginapp;

-- Create table
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

\echo 'Created table...';

-- Grant table permissions (safe to run multiple times)
ALTER TABLE IF EXISTS logins OWNER TO loginapp;
GRANT ALL PRIVILEGES ON TABLE logins TO loginapp;
GRANT USAGE, SELECT ON SEQUENCE logins_id_seq TO loginapp;

\echo 'Granted table permissions...';

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_logins ON logins (user_id, parsed_timestamp);
CREATE INDEX IF NOT EXISTS idx_device_logins ON logins (device_id, parsed_timestamp);
CREATE INDEX IF NOT EXISTS idx_ip_logins ON logins (ip, parsed_timestamp);

\echo 'Created indexes...';
\echo 'Database initialization complete!';