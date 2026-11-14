package discovery

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "local-info-scout"

type searchRequest struct {
	Query      string  `json:"query"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Radius     float64 `json:"radius"`
	Category   string  `json:"category,omitempty"`
	MinRating  float64 `json:"min_rating,omitempty"`
	MaxPrice   int     `json:"max_price,omitempty"`
	OpenNow    bool    `json:"open_now,omitempty"`
	Accessible bool    `json:"accessible,omitempty"`
}

type place struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	Category    string   `json:"category"`
	Distance    float64  `json:"distance"`
	Rating      float64  `json:"rating"`
	PriceLevel  int      `json:"price_level"`
	OpenNow     bool     `json:"open_now"`
	Photos      []string `json:"photos"`
	Description string   `json:"description"`
}

type searchResponse struct {
	Places  []place  `json:"places"`
	Sources []string `json:"sources"`
}

type categoriesReport struct {
	cliapp.ListReport
	Categories []string `json:"categories"`
}

type searchReport struct {
	cliapp.ListReport
	Request searchRequest `json:"request"`
	Places  []place       `json:"places"`
	Sources []string      `json:"sources"`
}

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Discovery",
		Commands: []cliapp.Command{
			{
				Name:        "categories",
				NeedsAPI:    true,
				Description: "List available search categories",
				Run: func(args []string) error {
					return runCategories(core, args)
				},
			},
			{
				Name:        "search",
				NeedsAPI:    true,
				Description: "Search for nearby places",
				Run: func(args []string) error {
					return runSearch(core, args)
				},
			},
		},
	}
}

func runCategories(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("categories", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/categories", nil)
	if err != nil {
		return err
	}

	var categories []string
	if err := json.Unmarshal(body, &categories); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := categoriesReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Available categories: %d", len(categories))},
			Results:        categories,
			RetrievalHints: []string{cliName + " search --query \"coffee shops\" --category restaurants"},
		},
		Categories: categories,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	query := fs.String("query", "", "Search query")
	lat := fs.Float64("lat", 40.7128, "Latitude")
	lon := fs.Float64("lon", -74.0060, "Longitude")
	radius := fs.Float64("radius", 5.0, "Search radius in miles")
	category := fs.String("category", "", "Category filter")
	minRating := fs.Float64("min-rating", 0, "Minimum rating filter")
	maxPrice := fs.Int("max-price", 0, "Maximum price level filter")
	openNow := fs.Bool("open-now", false, "Only include places that are open now")
	accessible := fs.Bool("accessible", false, "Only include accessible places")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	searchText := strings.TrimSpace(*query)
	if searchText == "" && fs.NArg() > 0 {
		searchText = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if searchText == "" {
		return fmt.Errorf("--query or a positional search term is required")
	}

	req := searchRequest{
		Query:      searchText,
		Lat:        *lat,
		Lon:        *lon,
		Radius:     *radius,
		Category:   strings.TrimSpace(*category),
		MinRating:  *minRating,
		MaxPrice:   *maxPrice,
		OpenNow:    *openNow,
		Accessible: *accessible,
	}

	body, err := core.Request("POST", "/search", nil, req)
	if err != nil {
		return err
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := searchReport{
		ListReport: cliapp.ListReport{
			Summary:        searchSummary(req, resp),
			Results:        searchRows(resp.Places),
			RetrievalHints: searchHints(req),
		},
		Request: req,
		Places:  resp.Places,
		Sources: resp.Sources,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func searchSummary(req searchRequest, resp searchResponse) []string {
	summary := []string{
		fmt.Sprintf("Query: %s", req.Query),
		fmt.Sprintf("Center: %.4f, %.4f", req.Lat, req.Lon),
		fmt.Sprintf("Radius: %.1f miles", req.Radius),
		fmt.Sprintf("Results: %d", len(resp.Places)),
	}
	if req.Category != "" {
		summary = append(summary, "Category: "+req.Category)
	}
	if len(resp.Sources) > 0 {
		summary = append(summary, "Sources: "+strings.Join(resp.Sources, ", "))
	}
	return summary
}

func searchRows(places []place) []string {
	if len(places) == 0 {
		return nil
	}
	rows := make([]string, 0, len(places))
	for i, item := range places {
		status := "closed"
		if item.OpenNow {
			status = "open"
		}
		row := fmt.Sprintf("%d. %s | %s | %.1f miles | %s", i+1, item.Name, item.Category, item.Distance, status)
		if item.Rating > 0 {
			row += fmt.Sprintf(" | rating %.1f", item.Rating)
		}
		if item.PriceLevel > 0 {
			row += " | " + strings.Repeat("$", item.PriceLevel)
		}
		if strings.TrimSpace(item.Address) != "" {
			row += " | " + item.Address
		}
		rows = append(rows, row)
		if strings.TrimSpace(item.Description) != "" {
			rows = append(rows, "   "+item.Description)
		}
	}
	return rows
}

func searchHints(req searchRequest) []string {
	hints := []string{
		cliName + " categories",
		cliName + " search --query \"" + req.Query + "\" --json",
	}
	if req.Category == "" {
		hints = append(hints, cliName+" search --query \""+req.Query+"\" --category restaurants")
	}
	return hints
}
