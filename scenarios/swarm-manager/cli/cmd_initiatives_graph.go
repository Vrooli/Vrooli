package main

// graph-show reads the materialized graph.json projection for an initiative.
// The projection is produced automatically by internal/graph on topology/
// backlog events; this command is the only CLI consumer, useful for agents
// building prompt context and for operators inspecting state at rest.

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type graphNodeDTO struct {
	ID       string `json:"id"`
	Kind     string `json:"kind,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Effort   string `json:"effort,omitempty"`
	Archived bool   `json:"archived,omitempty"`
}

type graphEdgeDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

type graphJSON struct {
	Initiative  string         `json:"initiative"`
	GeneratedAt string         `json:"generated_at,omitempty"`
	Nodes       []graphNodeDTO `json:"nodes"`
	Edges       []graphEdgeDTO `json:"edges"`
}

func (a *App) cmdInitiativesGraphShow(args []string) error {
	fs := flag.NewFlagSet("initiatives graph-show", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives graph-show --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/initiatives/"+name+"/files/graph.json", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	graph, err := decodeResponse[graphJSON](body)
	if err != nil {
		// Fall back to raw print in case the projection hasn't run yet and
		// the endpoint returned an error page or an unknown shape.
		fmt.Println(string(body))
		return nil
	}

	printSection("Graph")
	fmt.Printf("  Initiative:   %s\n", graph.Initiative)
	if graph.GeneratedAt != "" {
		fmt.Printf("  Generated at: %s\n", graph.GeneratedAt)
	}
	fmt.Printf("  Nodes:        %d\n", len(graph.Nodes))
	fmt.Printf("  Edges:        %d\n", len(graph.Edges))

	if len(graph.Nodes) > 0 {
		printSection("Nodes")
		for _, n := range graph.Nodes {
			archived := ""
			if n.Archived {
				archived = " [archived]"
			}
			fmt.Printf("  - %s — %s (status=%s, priority=%d, effort=%s)%s\n",
				n.ID, n.Title, n.Status, n.Priority, n.Effort, archived)
		}
	}
	if len(graph.Edges) > 0 {
		printSection("Edges")
		for _, e := range graph.Edges {
			kind := e.Kind
			if kind == "" {
				kind = "depends_on"
			}
			fmt.Printf("  - %s → %s (%s)\n", e.From, e.To, kind)
		}
	}
	// When nothing's materialized yet, make the hint obvious instead of
	// silently rendering an empty section.
	if len(graph.Nodes) == 0 && len(graph.Edges) == 0 {
		rest, _ := json.MarshalIndent(graph, "  ", "  ")
		printSection("Raw")
		fmt.Printf("  %s\n", string(rest))
	}
	return nil
}
