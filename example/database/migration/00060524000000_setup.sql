-- Auto-generated at Mon Apr  9 16:19:10 CEST 2018
-- Please do not change the name attributes

-- name: up

CREATE TABLE IF NOT EXISTS migrations (
 id          TEXT      NOT NULL PRIMARY KEY,
 description TEXT      NOT NULL,
 created_at  TIMESTAMP NOT NULL
);
-- name: down

DROP TABLE IF EXISTS migrations;