package parser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/emagjby/boot-blogagg/internal/database"
	"github.com/emagjby/boot-blogagg/internal/rss"
	"github.com/emagjby/boot-blogagg/internal/state"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Handlers map[string]func(*state.State, Command) error
}

func RegisterCommands() *Commands {
	cmds := &Commands{}
	cmds.register("login", CommandLogin)
	cmds.register("register", CommandRegister)
	cmds.register("reset", CommandReset)
	cmds.register("users", CommandListUsers)
	cmds.register("agg", CommandAgg)
	cmds.register("addfeed", middlewareLoggedIn(CommandAddFeed))
	cmds.register("feeds", CommandListFeeds)
	cmds.register("follow", middlewareLoggedIn(CommandFollow))
	cmds.register("unfollow", middlewareLoggedIn(CommandUnfollow))
	cmds.register("following", middlewareLoggedIn(CommandFollowing))
	cmds.register("browse", middlewareLoggedIn(CommandBrowse))
	return cmds
}

func middlewareLoggedIn(handler func(s *state.State, cmd Command, user database.User) error) func(*state.State, Command) error {
	return func(s *state.State, cmd Command) error {
		username, err := s.Config.GetUser()
		if err != nil {
			return fmt.Errorf("No user is currently logged in. Please login or register first.")
		}

		user, err := s.Db.GetUser(context.Background(), username)
		if err != nil {
			return fmt.Errorf("Failed to retrieve user information: %v", err)
		}

		return handler(s, cmd, user)
	}
}

func CommandFollowing(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) > 0 {
		return fmt.Errorf("Usage: gator following")
	}

	following, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("Failed to retrieve following feeds: %v", err)
	}

	if len(following) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}

	fmt.Printf("%s is following:\n", user.Name)
	for _, follow := range following {
		feed, err := s.Db.GetFeedById(context.Background(), follow.FeedID)
		if err != nil {
			fmt.Printf("- Failed to retrieve feed information for feed ID %s: %v\n", follow.FeedID, err)
			continue
		}
		fmt.Printf("- %s: %s\n", feed.Name, feed.Url)
	}

	return nil
}

func CommandFollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Usage: gator follow <feed_url>")
	}

	feedURL := cmd.Args[0]

	feed, err := s.Db.GetFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Failed to retrieve feed: %v", err)
	}

	id := uuid.New()
	created_at := time.Now()
	updated_at := time.Now()

	feedFollow := database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	if _, err := s.Db.CreateFeedFollow(context.Background(), feedFollow); err != nil {
		return fmt.Errorf("Failed to follow feed: %v", err)
	}

	fmt.Printf("You are now following feed '%s'.\n", feed.Name)
	return nil
}

func CommandUnfollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Usage: gator unfollow <feed_url>")
	}

	feedURL := cmd.Args[0]

	feed, err := s.Db.GetFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Failed to retrieve feed: %v", err)
	}

	if err := s.Db.DeleteFeedFollowForUser(context.Background(), database.DeleteFeedFollowForUserParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}); err != nil {
		return fmt.Errorf("Failed to unfollow feed: %v", err)
	}

	fmt.Printf("You have unfollowed feed '%s'.\n", feed.Name)
	return nil
}

func CommandListFeeds(s *state.State, cmd Command) error {
	feeds, err := s.Db.ListFeed(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to list feeds: %v", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("- %s: %s (by: %s)\n", feed.Name, feed.Url, feed.Username)
	}

	return nil
}

func CommandAddFeed(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("Usage: gator addfeed <name> <url>")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	id := uuid.New()
	created_at := time.Now()
	updated_at := time.Now()

	feed := database.CreateFeedParams{
		ID:        id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}

	if _, err := s.Db.CreateFeed(context.Background(), feed); err != nil {
		return fmt.Errorf("Failed to create feed: %v", err)
	}

	id = uuid.New()

	feedFollow := database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	if _, err := s.Db.CreateFeedFollow(context.Background(), feedFollow); err != nil {
		return fmt.Errorf("Failed to follow feed: %v", err)
	}

	fmt.Printf("Feed '%s' added successfully.\n", name)
	return nil
}

func CommandAgg(s *state.State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("Usage: gator agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("Invalid duration %q: %w", cmd.Args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			fmt.Printf("Error scraping feed: %v\n", err)
		}
	}
}

func scrapeFeeds(s *state.State) error {
	feed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No feeds found. Add one with: gator addfeed <name> <url>")
			return nil
		}
		return err
	}

	if err := s.Db.MarkFeedFetched(context.Background(), feed.ID); err != nil {
		return err
	}

	fmt.Printf("Fetching feed: %s (%s)\n", feed.Name, feed.Url)

	fetchedFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	postsCreated := 0
	for _, item := range fetchedFeed.Channel.Item {
		if item.Link == "" {
			fmt.Println("Skipping post with empty URL")
			continue
		}

		now := time.Now()
		postParams := database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Title:     item.Title,
			Url:       item.Link,
			FeedID:    feed.ID,
		}

		if item.Description != "" {
			postParams.Description = sql.NullString{String: item.Description, Valid: true}
		}

		if publishedAt, ok := parsePublishedAt(item.PubDate); ok {
			postParams.PublishedAt = sql.NullTime{Time: publishedAt, Valid: true}
		}

		if _, err := s.Db.CreatePost(context.Background(), postParams); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				continue
			}
			fmt.Printf("Failed to save post %q: %v\n", item.Title, err)
			continue
		}

		postsCreated++
	}

	fmt.Printf("Saved %d new posts from %s\n", postsCreated, feed.Name)

	return nil
}

func parsePublishedAt(raw string) (time.Time, bool) {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}

func CommandBrowse(s *state.State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) > 1 {
		return fmt.Errorf("Usage: gator browse [limit]")
	}

	if len(cmd.Args) == 1 {
		parsedLimit, err := strconv.Atoi(cmd.Args[0])
		if err != nil || parsedLimit <= 0 {
			return fmt.Errorf("Invalid limit: %s", cmd.Args[0])
		}
		limit = parsedLimit
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("Failed to get posts: %v", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found for followed feeds.")
		return nil
	}

	for _, post := range posts {
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format(time.RFC3339))
		}
		if post.Description.Valid {
			fmt.Printf("Description: %s\n", post.Description.String)
		}
		fmt.Println()
	}

	return nil
}

func CommandListUsers(s *state.State, cmd Command) error {
	users, err := s.Db.ListUsers(context.Background())
	if err != nil {
		return err
	}

	currentUser, _ := s.Config.GetUser()

	fmt.Println("Registered users:")
	for _, user := range users {
		if currentUser == user.Name {
			fmt.Printf("- %s (current)\n", user.Name)
			continue
		}
		fmt.Printf("- %s\n", user.Name)
	}

	return nil
}

func CommandLogin(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("Usage: gator login <username>")
	}

	name := cmd.Args[0]

	if _, err := s.Db.GetUser(context.Background(), name); err != nil {
		fmt.Printf("User %s does not exist. Please register first.\n", name)
		os.Exit(1)
	}

	if err := s.Config.SetUser(name); err != nil {
		return err
	}

	fmt.Printf("User has been set to %s\n", name)
	return nil
}

func CommandRegister(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("Usage: gator register <username>")
	}

	id := uuid.New()
	name := cmd.Args[0]
	created_at := time.Now()
	updated_at := time.Now()

	user := database.CreateUserParams{
		ID:        id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		Name:      name,
	}

	if _, err := s.Db.GetUser(context.Background(), name); err == nil {
		fmt.Printf("User %s already exists\n", cmd.Args[0])
		os.Exit(1)
	}

	s.Db.CreateUser(context.Background(), user)

	if err := s.Config.SetUser(name); err != nil {
		return err
	}

	fmt.Printf("User %s created. Identity set as %s", name, name)
	return nil
}

func CommandReset(s *state.State, cmd Command) error {
	if err := s.Db.DeleteFeedFollows(context.Background()); err != nil {
		return err
	}

	if err := s.Db.DeleteFeeds(context.Background()); err != nil {
		return err
	}

	if err := s.Db.DeleteUsers(context.Background()); err != nil {
		return err
	}

	fmt.Println("All users have been deleted.")
	return nil
}

func (c *Commands) register(name string, handler func(*state.State, Command) error) {
	if c.Handlers == nil {
		c.Handlers = make(map[string]func(*state.State, Command) error)
	}
	c.Handlers[name] = handler
}

func (c *Commands) Run(s *state.State, cmd Command) error {
	handler, exists := c.Handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("Unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}
