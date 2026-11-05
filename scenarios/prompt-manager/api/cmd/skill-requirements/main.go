package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"prompt-manager/internal/skills"
)

func main() {
	root := flag.String("root", "../../../store/skills/packs", "skill corpus root")
	flag.Parse()
	report, err := skills.SyncCorpusRequirements(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(data))
}
