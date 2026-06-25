package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogList(args []string) error {
	fs := flag.NewFlagSet("backlog list", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Comma-separated kinds to filter by")
	statusFlag := fs.String("status", "", "Comma-separated statuses to filter by")
	archivedFlag := fs.String("archived", "false", "Show archived items: true, false (default), or all")
	scenarioFlag := fs.String("scenario", "", "Comma-separated scenario names to filter by")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*kindFlag) != "" {
		query.Set("kinds", strings.TrimSpace(*kindFlag))
	}
	if strings.TrimSpace(*statusFlag) != "" {
		query.Set("statuses", strings.TrimSpace(*statusFlag))
	}
	if strings.TrimSpace(*archivedFlag) != "" {
		query.Set("archived", strings.TrimSpace(*archivedFlag))
	}
	if strings.TrimSpace(*scenarioFlag) != "" {
		query.Set("scenario", strings.TrimSpace(*scenarioFlag))
	}

	body, err := a.core.Get("/backlog", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListBacklogResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No backlog items found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "create", "--data", "'{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d backlog item(s)\n", len(response.Items))
	if kinds := strings.TrimSpace(query.Get("kinds")); kinds != "" {
		fmt.Printf("  Filtered kinds: %s\n", kinds)
	}
	if statuses := strings.TrimSpace(query.Get("statuses")); statuses != "" {
		fmt.Printf("  Filtered statuses: %s\n", statuses)
	}
	if scenario := strings.TrimSpace(query.Get("scenario")); scenario != "" {
		fmt.Printf("  Filtered scenarios: %s\n", scenario)
	}

	printSection("Results")
	for _, item := range response.Items {
		printBacklogListItem(item)
	}

	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "get", "--kind", "<kind>", "--name", "<name>"),
		cliCommand("backlog", "get", "--kind", first.Kind, "--name", first.Name),
		cliCommand("backlog", "files", "--kind", first.Kind, "--name", first.Name),
		cliCommand("backlog", "queue", "--kind", first.Kind, "--name", first.Name),
	})
	return nil
}

// printBacklogListItem renders a single backlog item entry for the list view.
func printBacklogListItem(item BacklogItem) {
	fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
	fmt.Printf("    Title: %s\n", item.Title)
	if len(item.Tags) > 0 {
		fmt.Printf("    Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if len(item.DependsOn) > 0 {
		fmt.Printf("    Depends on: %s\n", strings.Join(item.DependsOn, ", "))
	}
	if item.Initiative != "" {
		fmt.Printf("    Initiative: %s\n", item.Initiative)
	}
	if item.Effort != "" {
		fmt.Printf("    Effort: %s\n", item.Effort)
	}
	if len(item.AcceptanceAllow) > 0 {
		fmt.Printf("    Acceptance Allow: %s\n", strings.Join(item.AcceptanceAllow, ", "))
	}
	if len(item.AcceptanceDeny) > 0 {
		fmt.Printf("    Acceptance Deny: %s\n", strings.Join(item.AcceptanceDeny, ", "))
	}
	if len(item.Creates) > 0 {
		fmt.Printf("    Creates: %s\n", strings.Join(item.Creates, ", "))
	}
	fmt.Println()
}

func (a *App) cmdBacklogGet(args []string) error {
	fs := flag.NewFlagSet("backlog get", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog get --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/backlog/"+kind+"/"+name, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Summary")
	fmt.Printf("  %s/%s (%s)\n", item.Kind, item.Name, item.Status)

	printSection("Details")
	fmt.Printf("  Name: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Title: %s\n", item.Title)
	fmt.Printf("  Description: %s\n", item.Description)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if len(item.DependsOn) > 0 {
		fmt.Printf("  Depends On: %s\n", strings.Join(item.DependsOn, ", "))
	}
	if item.Initiative != "" {
		fmt.Printf("  Initiative: %s\n", item.Initiative)
	}
	if item.Effort != "" {
		fmt.Printf("  Effort: %s\n", item.Effort)
	}
	if len(item.AcceptanceAllow) > 0 {
		fmt.Printf("  Acceptance Allow: %s\n", strings.Join(item.AcceptanceAllow, ", "))
	}
	if len(item.AcceptanceDeny) > 0 {
		fmt.Printf("  Acceptance Deny: %s\n", strings.Join(item.AcceptanceDeny, ", "))
	}
	if len(item.Creates) > 0 {
		fmt.Printf("  Creates: %s\n", strings.Join(item.Creates, ", "))
	}
	fmt.Printf("  Created: %s\n", item.Created)
	fmt.Printf("  Updated: %s\n", item.Updated)

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "update", "--kind", item.Kind, "--name", item.Name, "--data", "'{\"status\":\"ready\"}'"),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) cmdBacklogCreate(args []string) error {
	fs := flag.NewFlagSet("backlog create", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	var attachFlags stringSlice
	fs.Var(&attachFlags, "attach", "Attach file as destination=source (repeatable)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: backlog create --data JSON [--attach DEST=SRC] [--json]\n\nExample:\n  backlog create --data '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	// Auto-detect spawn provenance from environment. The agent process
	// inherits VROOLI_SPAWN_SOURCE from the parent (set by swarm-manager
	// when spawning via agent-manager). Inject it into the create payload
	// so the API can persist the link automatically.
	if spawnSource := os.Getenv("VROOLI_SPAWN_SOURCE"); spawnSource != "" {
		payload = injectJSONField(payload, "spawned_from", spawnSource)
	}

	var req CreateBacklogRequest
	if err := decodeJSONStrict(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" || req.Kind == "" {
		return fmt.Errorf("name, title, and kind are required fields")
	}

	var body []byte
	if len(attachFlags) > 0 {
		body, err = a.createBacklogMultipart(payload, attachFlags)
	} else {
		body, err = a.core.Request("POST", "/backlog", nil, payload)
	}
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Created backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "files", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) createBacklogMultipart(itemPayload []byte, attachments []string) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("item", string(itemPayload)); err != nil {
		return nil, fmt.Errorf("write item part: %w", err)
	}

	type manifestEntry struct {
		Field string `json:"field"`
		Path  string `json:"path"`
	}
	manifest := struct {
		Files []manifestEntry `json:"files"`
	}{Files: make([]manifestEntry, 0, len(attachments))}

	for index, raw := range attachments {
		dest, src, ok := strings.Cut(raw, "=")
		dest = strings.TrimSpace(dest)
		src = strings.TrimSpace(src)
		if !ok || dest == "" || src == "" {
			return nil, fmt.Errorf("invalid --attach %q: expected destination=source", raw)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("read attachment %q: %w", src, err)
		}
		field := fmt.Sprintf("file_%d", index)
		part, err := writer.CreateFormFile(field, filepath.Base(dest))
		if err != nil {
			return nil, fmt.Errorf("create file part %q: %w", dest, err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("write file part %q: %w", dest, err)
		}
		manifest.Files = append(manifest.Files, manifestEntry{Field: field, Path: dest})
	}

	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode file manifest: %w", err)
	}
	if err := writer.WriteField("files_manifest", string(manifestPayload)); err != nil {
		return nil, fmt.Errorf("write manifest part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}
	return a.requestMultipart("POST", "/backlog", body.Bytes(), writer.FormDataContentType())
}

func (a *App) cmdBacklogUpdate(args []string) error {
	fs := flag.NewFlagSet("backlog update", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: backlog update --kind KIND --name NAME --data JSON [--json]\n\nExample:\n  backlog update --kind idea --name my-idea --data '{\"status\":\"ready\"}'\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var update UpdateBacklogRequest
	if err := decodeJSONStrict(payload, &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if update.Empty() {
		return fmt.Errorf("at least one field must be provided")
	}

	requestBody, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update payload: %w", err)
	}

	body, err := a.core.Request("PATCH", "/backlog/"+kind+"/"+name, nil, json.RawMessage(requestBody))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Updated backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) cmdBacklogDelete(args []string) error {
	fs := flag.NewFlagSet("backlog delete", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog delete --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Request("DELETE", "/backlog/"+kind+"/"+name, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Printf("  Deleted backlog item: %s/%s\n", kind, name)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "list"),
		cliCommand("backlog", "create", "--data", "'{\"name\":\"new-item\",\"title\":\"New Item\",\"kind\":\"idea\"}'"),
	})
	return nil
}
