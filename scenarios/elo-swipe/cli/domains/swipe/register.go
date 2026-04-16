package swipe

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"elo-swipe/cli/internal/client"
	"elo-swipe/cli/internal/render"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "elo-swipe"

func Register(api *client.Client) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "swipe",
		Description: "Interactive ranking flows",
		Subcommands: []cliapp.Command{
			{Name: "run", NeedsAPI: true, Description: "Start interactive swipe mode for a list", Run: func(args []string) error { return run(api, args) }},
		},
	}
}

func run(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("swipe run", flag.ContinueOnError)
	listID := fs.String("list", "", "List ID (required)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *listID == "" {
		return fmt.Errorf("--list is required")
	}

	fmt.Println("Interactive Swipe Mode")
	fmt.Println("Enter A for the first item, B for the second item, S to skip, or Q to quit.")

	reader := bufio.NewReader(os.Stdin)
	for {
		next, err := api.NextComparison(*listID)
		if err != nil {
			return err
		}
		if next == nil {
			fmt.Printf("Ranking complete. View results with: %s rankings show --list %s\n", cliName, *listID)
			return nil
		}

		fmt.Println()
		fmt.Printf("Progress: %d / %d\n", next.Progress.Completed, next.Progress.Total)
		fmt.Printf("A: %s\n", render.ItemLabel(next.ItemA.Content))
		fmt.Printf("B: %s\n", render.ItemLabel(next.ItemB.Content))
		fmt.Print("Choice (A/B/S/Q): ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read choice: %w", err)
		}
		switch strings.ToUpper(strings.TrimSpace(line)) {
		case "A":
			if _, err := api.CreateComparison(*listID, next.ItemA.ID, next.ItemB.ID); err != nil {
				return err
			}
			fmt.Println("Recorded choice: A")
		case "B":
			if _, err := api.CreateComparison(*listID, next.ItemB.ID, next.ItemA.ID); err != nil {
				return err
			}
			fmt.Println("Recorded choice: B")
		case "S":
			fmt.Println("Skipped")
		case "Q":
			fmt.Println("Goodbye.")
			return nil
		default:
			fmt.Println("Invalid choice. Enter A, B, S, or Q.")
		}
	}
}
