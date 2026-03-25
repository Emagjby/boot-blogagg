package state

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/emagjby/boot-blogagg/internal/config"
	"github.com/emagjby/boot-blogagg/internal/database"
)

type State struct {
	Config *config.Config
	Db     *database.Queries
}

func BuildState(cfg config.Config) *State {
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	return &State{
		Config: &cfg,
		Db:     dbQueries,
	}
}
