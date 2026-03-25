# gator (boot-blogagg)

`gator` is a Go CLI that lets you follow RSS feeds, continuously scrape posts, and browse recent posts from the feeds you follow.

## Requirements

- Go (1.22+ recommended)
- PostgreSQL (local or remote)

## Install the CLI

Install from source with `go install`:

```bash
go install github.com/emagjby/boot-blogagg@latest
```

This compiles a static binary into your Go bin directory (usually `~/go/bin`).

Note: with the current module layout, the installed binary name is `boot-blogagg`.
If you want to run it as `gator`, create an alias or rename the binary.

Examples:

```bash
# one-time rename
mv ~/go/bin/boot-blogagg ~/go/bin/gator

# or shell alias
alias gator="$HOME/go/bin/boot-blogagg"
```

After `go build` or `go install`, you can run the binary directly without `go run`.

## Database setup

1. Create a PostgreSQL database (example name: `gator`).
2. Run migrations (Goose):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

Adjust username/password/host/database in the connection string as needed.

## Config file

Create `~/.gatorconfig.json`:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

- `db_url`: PostgreSQL connection string
- `current_user_name`: managed by the CLI after login/register

## Quick start

```bash
gator register yourname
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"
gator follow https://news.ycombinator.com/rss
gator agg 1m
```

Run `agg` in one terminal so it keeps collecting posts in the background.

In another terminal:

```bash
gator browse
gator browse 10
```

## Useful commands

- `register <username>`: create a user and set as current
- `login <username>`: switch current user
- `users`: list users
- `addfeed <name> <url>`: add a feed and auto-follow it
- `feeds`: list feeds
- `follow <feed_url>` / `unfollow <feed_url>`: manage follows
- `following`: list feeds current user follows
- `agg <time_between_reqs>`: continuously scrape feeds (e.g. `1s`, `30s`, `5m`)
- `browse [limit]`: show latest posts from followed feeds (default: `2`)
- `reset`: delete all data (users, feeds, follows, posts)

## Development

Use `go run .` during development. Example:

```bash
go run . agg 30s
```

For production usage, run the compiled binary (`gator` or `boot-blogagg`).
