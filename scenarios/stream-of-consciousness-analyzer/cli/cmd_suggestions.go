package main

import "fmt"

// --- Suggestion / Provider Commands ---

func (a *App) cmdProviderList(args []string) error {
	_, jsonOut, err := a.cmdFlags("provider list", args)
	if err != nil {
		return err
	}
	return a.getResource("/providers", jsonOut, func(body []byte) error {
		var providers []struct {
			Name     string `json:"name"`
			Active   bool   `json:"active"`
			Fallback bool   `json:"fallback"`
		}
		if err := unmarshalBody(body, &providers); err != nil {
			return err
		}
		if len(providers) == 0 {
			fmt.Println("No LLM providers configured.")
			return nil
		}
		for _, p := range providers {
			status := "inactive"
			if p.Active && p.Fallback {
				status = "active (fallback)"
			} else if p.Active {
				status = "active"
			}
			fmt.Printf("%-20s  %s\n", p.Name, status)
		}
		return nil
	})
}

func (a *App) cmdSuggestionGenerate(args []string) error {
	fs, jsonOut, err := a.cmdFlags("suggestion generate", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "suggestion generate <scheme-id> [--json]"); err != nil {
		return err
	}
	return a.postResource("/schemes/"+fs.Arg(0)+"/suggestions", nil, jsonOut, func(body []byte) error {
		var resp struct {
			Suggestions []struct {
				Label      string  `json:"label"`
				Confidence float64 `json:"confidence"`
			} `json:"suggestions"`
			Provider string `json:"provider"`
		}
		if err := unmarshalBody(body, &resp); err != nil {
			return err
		}
		fmt.Printf("Provider: %s\n", resp.Provider)
		fmt.Printf("Suggestions: %d\n", len(resp.Suggestions))
		for _, s := range resp.Suggestions {
			fmt.Printf("  - %s (confidence: %.2f)\n", s.Label, s.Confidence)
		}
		return nil
	})
}
