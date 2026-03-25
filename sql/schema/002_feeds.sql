--
-- name: The name of the feed (like "The Changelog, or "The Boot.dev Blog")
-- url: The URL of the feed
-- user_id: The ID of the user who added this feed
-- +goose Up
CREATE TABLE feeds(
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id)
ON DELETE CASCADE
);
-- +goose Down
DROP TABLE feeds;
