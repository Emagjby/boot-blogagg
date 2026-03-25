package parser

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/emagjby/boot-blogagg/internal/database"
	"github.com/emagjby/boot-blogagg/internal/rss"
	"github.com/emagjby/boot-blogagg/internal/state"
	"github.com/google/uuid"
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
	cmds.register("addfeed", CommandAddFeed)
	cmds.register("feeds", CommandListFeeds)
	return cmds
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

func CommandAddFeed(s *state.State, cmd Command) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("Usage: gator addfeed <name> <url>")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	username, err := s.Config.GetUser()
	if err != nil {
		return fmt.Errorf("No user is currently logged in. Please login or register first.")
	}

	user_id, err := s.Db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("Failed to retrieve user information: %v", err)
	}

	id := uuid.New()
	created_at := time.Now()
	updated_at := time.Now()

	feed := database.CreateFeedParams{
		ID:        id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		Name:      name,
		Url:       url,
		UserID:    user_id.ID,
	}

	if _, err := s.Db.CreateFeed(context.Background(), feed); err != nil {
		return fmt.Errorf("Failed to create feed: %v", err)
	}

	fmt.Printf("Feed '%s' added successfully.\n", name)
	return nil
}

func CommandAgg(s *state.State, cmd Command) error {
	agg, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Printf("Title: %s\n", agg.Channel.Title)
	fmt.Printf("Link: %s\n", agg.Channel.Link)
	fmt.Printf("Description: %s\n", agg.Channel.Description)
	fmt.Println("Items:")
	for _, item := range agg.Channel.Item {
		fmt.Printf("- Title: %s\n", item.Title)
		fmt.Printf("  Link: %s\n", item.Link)
		fmt.Printf("  Description: %s\n", item.Description)
		fmt.Printf("  PubDate: %s\n", item.PubDate)
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
