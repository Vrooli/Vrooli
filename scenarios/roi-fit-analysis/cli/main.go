package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start roi-fit-analysis

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

    if len(os.Args) < 2 {
        fmt.Println("ROI Fit Analysis CLI")
        fmt.Println("Usage: roi-fit <command> [options]")
        fmt.Println("\nCommands:")
        fmt.Println("  analyze <idea> <budget>  - Analyze ROI for an idea")
        fmt.Println("  list                     - List all opportunities")
        fmt.Println("  report                   - Generate analysis report")
        os.Exit(1)
    }

    apiURL := os.Getenv("API_URL")
    if apiURL == "" {
        apiURL = "http://localhost:3000"
    }

    command := os.Args[1]

    switch command {
    case "analyze":
        if len(os.Args) < 4 {
            fmt.Println("Error: analyze requires idea and budget")
            os.Exit(1)
        }
        analyzeIdea(apiURL, os.Args[2], os.Args[3])
    
    case "list":
        listOpportunities(apiURL)
    
    case "report":
        generateReport(apiURL)
    
    default:
        fmt.Printf("Unknown command: %s\n", command)
        os.Exit(1)
    }
}

func analyzeIdea(apiURL, idea, budget string) {
    payload := map[string]interface{}{
        "idea":     idea,
        "budget":   budget,
        "timeline": "12 months",
        "skills":   []string{"development", "marketing"},
    }

    jsonData, _ := json.Marshal(payload)
    resp, err := http.Post(apiURL+"/analyze", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    
    var result map[string]interface{}
    json.Unmarshal(body, &result)
    
    fmt.Println("\n=== ROI Analysis Results ===")
    if analysis, ok := result["analysis"].(map[string]interface{}); ok {
        fmt.Printf("ROI Score: %.1f/100\n", analysis["roi_score"])
        fmt.Printf("Recommendation: %s\n", analysis["recommendation"])
        fmt.Printf("Payback Period: %v months\n", analysis["payback_months"])
        fmt.Printf("Risk Level: %s\n", analysis["risk_level"])
        fmt.Printf("Market Size: %s\n", analysis["market_size"])
    }
}

func listOpportunities(apiURL string) {
    resp, err := http.Get(apiURL + "/opportunities")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    
    var opportunities []map[string]interface{}
    json.Unmarshal(body, &opportunities)
    
    fmt.Println("\n=== Investment Opportunities ===")
    for i, opp := range opportunities {
        fmt.Printf("\n%d. %s\n", i+1, opp["name"])
        fmt.Printf("   ROI Score: %.1f\n", opp["roi_score"])
        fmt.Printf("   Investment: $%.0f\n", opp["investment"])
        fmt.Printf("   Payback: %v months\n", opp["payback_months"])
        fmt.Printf("   Risk: %s\n", opp["risk_level"])
    }
}

func generateReport(apiURL string) {
    resp, err := http.Get(apiURL + "/reports")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    
    var report map[string]interface{}
    json.Unmarshal(body, &report)
    
    fmt.Println("\n=== ROI Analysis Report ===")
    if summary, ok := report["summary"].(map[string]interface{}); ok {
        fmt.Printf("Total Analyzed: %.0f\n", summary["total_analyzed"])
        fmt.Printf("High ROI Count: %.0f\n", summary["high_roi_count"])
        fmt.Printf("Average ROI: %.1f%%\n", summary["average_roi"])
        fmt.Printf("Best Opportunity: %s\n", summary["best_opportunity"])
    }
    
    if recs, ok := report["recommendations"].([]interface{}); ok {
        fmt.Println("\nRecommendations:")
        for i, rec := range recs {
            fmt.Printf("%d. %s\n", i+1, rec)
        }
    }
}