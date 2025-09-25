package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
)

const VERSION = "1.0.0"

var apiURL string

func init() {
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		// Try to find the actual running port from the API process
		apiPort = "19294" // Default to current running port
	}
	apiURL = fmt.Sprintf("http://localhost:%s/api/v1", apiPort)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start elo-swipe

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Define commands
	var (
		statusCmd      = flag.NewFlagSet("status", flag.ExitOnError)
		listCmd        = flag.NewFlagSet("list", flag.ExitOnError)
		createCmd      = flag.NewFlagSet("create-list", flag.ExitOnError)
		rankingsCmd    = flag.NewFlagSet("rankings", flag.ExitOnError)
		compareCmd     = flag.NewFlagSet("compare", flag.ExitOnError)
		swipeCmd       = flag.NewFlagSet("swipe", flag.ExitOnError)
	)

	// Status flags
	statusJSON := statusCmd.Bool("json", false, "Output as JSON")
	statusVerbose := statusCmd.Bool("verbose", false, "Verbose output")

	// Create list flags
	createName := createCmd.String("name", "", "List name (required)")
	createDesc := createCmd.String("description", "", "List description")
	createItems := createCmd.String("items-file", "", "JSON file containing items (required)")

	// Rankings flags
	rankingsList := rankingsCmd.String("list", "", "List ID (required)")
	rankingsFormat := rankingsCmd.String("format", "table", "Output format: table, json, csv")
	rankingsTop := rankingsCmd.Int("top", 0, "Show only top N items")

	// Compare flags
	compareList := compareCmd.String("list", "", "List ID (required)")
	compareWinner := compareCmd.String("winner", "", "Winner item ID")
	compareLoser := compareCmd.String("loser", "", "Loser item ID")

	// Swipe flags
	swipeList := swipeCmd.String("list", "", "List ID (required)")

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "help", "--help", "-h":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printHelp()
		}

	case "version", "--version", "-v":
		fmt.Printf("elo-swipe version %s\n", VERSION)

	case "status":
		statusCmd.Parse(os.Args[2:])
		showStatus(*statusJSON, *statusVerbose)

	case "list":
		listCmd.Parse(os.Args[2:])
		listLists()

	case "create-list":
		createCmd.Parse(os.Args[2:])
		if *createName == "" || *createItems == "" {
			fmt.Println("Error: --name and --items-file are required")
			createCmd.PrintDefaults()
			os.Exit(1)
		}
		createList(*createName, *createDesc, *createItems)

	case "rankings":
		rankingsCmd.Parse(os.Args[2:])
		if *rankingsList == "" {
			fmt.Println("Error: --list is required")
			rankingsCmd.PrintDefaults()
			os.Exit(1)
		}
		showRankings(*rankingsList, *rankingsFormat, *rankingsTop)

	case "compare":
		compareCmd.Parse(os.Args[2:])
		if *compareList == "" {
			fmt.Println("Error: --list is required")
			compareCmd.PrintDefaults()
			os.Exit(1)
		}
		submitComparison(*compareList, *compareWinner, *compareLoser)

	case "swipe":
		swipeCmd.Parse(os.Args[2:])
		if *swipeList == "" {
			fmt.Println("Error: --list is required")
			swipeCmd.PrintDefaults()
			os.Exit(1)
		}
		interactiveSwipe(*swipeList)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Elo Swipe CLI - Universal ranking engine")
	fmt.Println("\nUsage: elo-swipe <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  status           Show operational status")
	fmt.Println("  list             List all ranking lists")
	fmt.Println("  create-list      Create a new ranking list")
	fmt.Println("  rankings         Display current rankings for a list")
	fmt.Println("  compare          Submit a comparison result")
	fmt.Println("  swipe            Interactive swipe interface (CLI)")
	fmt.Println("  help             Show this help message")
	fmt.Println("  version          Show version information")
	fmt.Println("\nUse 'elo-swipe help <command>' for more information about a command.")
}

func printCommandHelp(command string) {
	switch command {
	case "status":
		fmt.Println("Show operational status and database health")
		fmt.Println("\nUsage: elo-swipe status [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --json       Output as JSON")
		fmt.Println("  --verbose    Show detailed information")

	case "create-list":
		fmt.Println("Create a new ranking list")
		fmt.Println("\nUsage: elo-swipe create-list [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --name <name>           List name (required)")
		fmt.Println("  --description <desc>    List description")
		fmt.Println("  --items-file <file>     JSON file containing items (required)")
		fmt.Println("\nExample JSON file format:")
		fmt.Println(`  ["Item 1", "Item 2", "Item 3"]`)
		fmt.Println(`  or`)
		fmt.Println(`  [{"title": "Item 1"}, {"title": "Item 2"}]`)

	case "rankings":
		fmt.Println("Display current rankings for a list")
		fmt.Println("\nUsage: elo-swipe rankings [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --list <id>         List ID (required)")
		fmt.Println("  --format <format>   Output format: table, json, csv (default: table)")
		fmt.Println("  --top <n>           Show only top N items")

	default:
		fmt.Printf("No help available for command: %s\n", command)
	}
}

func showStatus(asJSON, verbose bool) {
	resp, err := http.Get(apiURL + "/health")
	if err != nil {
		if asJSON {
			fmt.Printf(`{"status":"error","message":"%s"}`+"\n", err.Error())
		} else {
			fmt.Printf("❌ API is not responding: %s\n", err.Error())
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if asJSON {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	} else {
		if resp.StatusCode == 200 {
			fmt.Println("✅ Elo Swipe is operational")
			if verbose {
				fmt.Printf("API endpoint: %s\n", apiURL)
				fmt.Printf("Version: %s\n", VERSION)
			}
		} else {
			fmt.Printf("⚠️  API returned status code: %d\n", resp.StatusCode)
		}
	}
}

func listLists() {
	resp, err := http.Get(apiURL + "/lists")
	if err != nil {
		fmt.Printf("Error fetching lists: %s\n", err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	var lists []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		fmt.Printf("Error parsing response: %s\n", err.Error())
		os.Exit(1)
	}

	if len(lists) == 0 {
		fmt.Println("No lists found. Create one with 'elo-swipe create-list'")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tITEMS\tCREATED")
	fmt.Fprintln(w, "---\t----\t-----\t-------")

	for _, list := range lists {
		id := list["id"].(string)
		if len(id) > 8 {
			id = id[:8] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\n",
			id,
			list["name"],
			list["item_count"],
			list["created_at"])
	}
	w.Flush()
}

func createList(name, description, itemsFile string) {
	// Read items file
	data, err := ioutil.ReadFile(itemsFile)
	if err != nil {
		fmt.Printf("Error reading items file: %s\n", err.Error())
		os.Exit(1)
	}

	// Parse items
	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		fmt.Printf("Error parsing items file: %s\n", err.Error())
		os.Exit(1)
	}

	// Convert to API format
	items := make([]map[string]json.RawMessage, len(rawItems))
	for i, item := range rawItems {
		items[i] = map[string]json.RawMessage{"content": item}
	}

	// Create request
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"items":       items,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL+"/lists", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error creating list: %s\n", err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("✅ List created successfully!\n")
	fmt.Printf("List ID: %s\n", result["list_id"])
	fmt.Printf("Items: %.0f\n", result["item_count"])
	fmt.Printf("\nStart ranking with: elo-swipe swipe --list %s\n", result["list_id"])
}

func showRankings(listID, format string, top int) {
	resp, err := http.Get(fmt.Sprintf("%s/lists/%s/rankings", apiURL, listID))
	if err != nil {
		fmt.Printf("Error fetching rankings: %s\n", err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string][]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %s\n", err.Error())
		os.Exit(1)
	}

	rankings := result["rankings"]
	if top > 0 && top < len(rankings) {
		rankings = rankings[:top]
	}

	switch format {
	case "json":
		output, _ := json.MarshalIndent(rankings, "", "  ")
		fmt.Println(string(output))

	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"Rank", "Item", "Elo Rating", "Confidence"})
		for _, item := range rankings {
			w.Write([]string{
				fmt.Sprintf("%.0f", item["rank"]),
				formatItem(item["item"]),
				fmt.Sprintf("%.0f", item["elo_rating"]),
				fmt.Sprintf("%.1f%%", item["confidence"].(float64)*100),
			})
		}
		w.Flush()

	default: // table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RANK\tITEM\tRATING\tCONFIDENCE")
		fmt.Fprintln(w, "----\t----\t------\t----------")

		for _, item := range rankings {
			fmt.Fprintf(w, "%.0f\t%s\t%.0f\t%.1f%%\n",
				item["rank"],
				formatItem(item["item"]),
				item["elo_rating"],
				item["confidence"].(float64)*100)
		}
		w.Flush()
	}
}

func formatItem(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]interface{}:
		if title, ok := v["title"].(string); ok {
			return title
		}
		if name, ok := v["name"].(string); ok {
			return name
		}
		data, _ := json.Marshal(v)
		return string(data)
	default:
		data, _ := json.Marshal(v)
		str := string(data)
		// Remove quotes if it's a simple string
		if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
			str = str[1 : len(str)-1]
		}
		return str
	}
}

func submitComparison(listID, winnerID, loserID string) {
	if winnerID == "" || loserID == "" {
		// Get next comparison if IDs not provided
		resp, err := http.Get(fmt.Sprintf("%s/lists/%s/next-comparison", apiURL, listID))
		if err != nil {
			fmt.Printf("Error getting next comparison: %s\n", err.Error())
			os.Exit(1)
		}
		defer resp.Body.Close()

		var comparison map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&comparison); err != nil {
			fmt.Println("No more comparisons needed")
			return
		}

		fmt.Printf("Which is better?\n")
		fmt.Printf("A: %s\n", formatItem(comparison["item_a"].(map[string]interface{})["content"]))
		fmt.Printf("B: %s\n", formatItem(comparison["item_b"].(map[string]interface{})["content"]))
		fmt.Printf("\nEnter A or B: ")

		var choice string
		fmt.Scanln(&choice)

		if strings.ToUpper(choice) == "A" {
			winnerID = comparison["item_a"].(map[string]interface{})["id"].(string)
			loserID = comparison["item_b"].(map[string]interface{})["id"].(string)
		} else {
			winnerID = comparison["item_b"].(map[string]interface{})["id"].(string)
			loserID = comparison["item_a"].(map[string]interface{})["id"].(string)
		}
	}

	// Submit comparison
	reqBody := map[string]string{
		"list_id":   listID,
		"winner_id": winnerID,
		"loser_id":  loserID,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL+"/comparisons", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error submitting comparison: %s\n", err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Println("✅ Comparison recorded")
}

func interactiveSwipe(listID string) {
	fmt.Println("Interactive Swipe Mode")
	fmt.Println("Press A for left, B for right, S to skip, Q to quit")
	fmt.Println("----------------------------------------")

	for {
		// Get next comparison
		resp, err := http.Get(fmt.Sprintf("%s/lists/%s/next-comparison", apiURL, listID))
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			break
		}

		if resp.StatusCode == 204 {
			fmt.Println("\n✅ Ranking complete! View results with: elo-swipe rankings --list " + listID)
			break
		}

		var comparison map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&comparison); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		// Show comparison
		progress := comparison["progress"].(map[string]interface{})
		fmt.Printf("\nProgress: %.0f/%.0f comparisons\n\n", 
			progress["completed"], progress["total"])

		itemA := comparison["item_a"].(map[string]interface{})
		itemB := comparison["item_b"].(map[string]interface{})

		fmt.Printf("[A] %s\n", formatItem(itemA["content"]))
		fmt.Printf("    vs\n")
		fmt.Printf("[B] %s\n", formatItem(itemB["content"]))
		fmt.Printf("\nChoice (A/B/S/Q): ")

		var choice string
		fmt.Scanln(&choice)

		switch strings.ToUpper(choice) {
		case "A":
			submitComparison(listID, itemA["id"].(string), itemB["id"].(string))
		case "B":
			submitComparison(listID, itemB["id"].(string), itemA["id"].(string))
		case "S":
			fmt.Println("Skipped")
			continue
		case "Q":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice. Please enter A, B, S, or Q")
		}
	}
}