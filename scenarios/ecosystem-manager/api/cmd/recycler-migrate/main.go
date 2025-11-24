package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

var statuses = []string{
	"pending",
	"in-progress",
	"completed",
	"completed-finalized",
	"failed",
	"failed-blocked",
}

func main() {
	queueDirFlag := flag.String("queue-dir", "", "Path to the ecosystem-manager queue directory")
	flag.Parse()

	queueDir, err := resolveQueueDir(*queueDirFlag)
	if err != nil {
		log.Fatalf("Unable to resolve queue directory: %v", err)
	}

	storage := tasks.NewStorage(queueDir)

	var total, updated int
	for _, status := range statuses {
		items, err := storage.GetQueueItems(status)
		if err != nil {
			log.Printf("Warning: failed to read %s tasks: %v", status, err)
			continue
		}

		for _, item := range items {
			total++
			mutated := false

			if item.ConsecutiveCompletionClaims < 0 {
				item.ConsecutiveCompletionClaims = 0
				mutated = true
			}
			if item.ConsecutiveFailures < 0 {
				item.ConsecutiveFailures = 0
				mutated = true
			}

			switch status {
			case "completed-finalized", "failed-blocked":
				if item.ProcessorAutoRequeue {
					item.ProcessorAutoRequeue = false
					mutated = true
				}
			default:
				if !item.ProcessorAutoRequeue {
					item.ProcessorAutoRequeue = true
					mutated = true
				}
			}

			if mutated {
				if err := storage.SaveQueueItem(item, status); err != nil {
					log.Printf("Failed to save task %s in %s: %v", item.ID, status, err)
					continue
				}
				updated++
			}
		}
	}

	fmt.Printf("Recycler migration complete. Processed %d tasks, updated %d.\n", total, updated)
}

func resolveQueueDir(userProvided string) (string, error) {
	if userProvided != "" {
		return userProvided, nil
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, "scenarios", "ecosystem-manager", "queue"), nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not locate repository root (missing .vrooli)")
}
