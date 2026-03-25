package main

import (
	_ "github.com/lib/pq"

	"fmt"
	"os"

	"github.com/emagjby/boot-blogagg/internal/config"
	"github.com/emagjby/boot-blogagg/internal/parser"
	"github.com/emagjby/boot-blogagg/internal/state"
)

func main() {
	cfg, err := config.ReadJsonConfig()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	app := state.BuildState(cfg)
	cmds := parser.RegisterCommands()

	if len(os.Args) < 2 {
		fmt.Println("Error: not enough arguments")
		os.Exit(1)
	}

	cmd := parser.Command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	if err := cmds.Run(app, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
