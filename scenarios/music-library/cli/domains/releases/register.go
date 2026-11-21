package releases

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "releases"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	list := cliapp.Action(func(ctx cliapp.OperationContext) (map[string]any, error) {
		query := url.Values{}
		if limit := ctx.Flag("limit"); limit != "" {
			query.Set("limit", limit)
		}
		if cursor := ctx.Flag("cursor"); cursor != "" {
			query.Set("cursor", cursor)
		}
		return request(core, "GET", "/releases", query, nil)
	}, func(_ cliapp.OperationContext, result map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Release catalog page:", pretty(result)}}
	})

	create := cliapp.Action(func(ctx cliapp.OperationContext) (map[string]any, error) {
		path := ctx.Flag("file")
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("read edition file: %w", err)
		}
		if info.Size() < 1 || info.Size() > 1<<20 {
			return nil, fmt.Errorf("edition file must be between 1 byte and 1 MiB")
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var edition map[string]any
		if err := json.Unmarshal(body, &edition); err != nil {
			return nil, fmt.Errorf("edition file must contain a JSON object: %w", err)
		}
		return request(core, "POST", "/releases", nil, edition)
	}, func(_ cliapp.OperationContext, result map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Saved immutable release draft:", pretty(result)}}
	})

	get := cliapp.Action(func(ctx cliapp.OperationContext) (map[string]any, error) {
		query := url.Values{}
		if revision := ctx.Flag("revision"); revision != "" {
			query.Set("revision", revision)
		}
		return request(core, "GET", "/releases/"+url.PathEscape(ctx.Positional("release_id")), query, nil)
	}, func(_ cliapp.OperationContext, result map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Release revision and eligibility:", pretty(result)}}
	})

	review := cliapp.Action(func(ctx cliapp.OperationContext) (map[string]any, error) {
		var revision int
		if _, err := fmt.Sscan(ctx.Flag("revision"), &revision); err != nil || revision < 1 {
			return nil, fmt.Errorf("revision must be a positive integer")
		}
		if ctx.Flag("all-tracks-heard") != "yes" {
			return nil, fmt.Errorf("pass --all-tracks-heard yes only after listening to every exact-hash track in the pinned revision")
		}
		editionResponse, err := request(core, "GET", "/releases/"+url.PathEscape(ctx.Positional("release_id")), url.Values{"revision": []string{fmt.Sprint(revision)}}, nil)
		if err != nil {
			return nil, err
		}
		edition, ok := editionResponse["edition"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("release response does not contain an edition")
		}
		trackRows, ok := edition["tracks"].([]any)
		if !ok || len(trackRows) == 0 {
			return nil, fmt.Errorf("release response does not contain ordered track hashes")
		}
		trackHashes := make([]string, 0, len(trackRows))
		for _, raw := range trackRows {
			track, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("release response has an invalid track occurrence")
			}
			hash, ok := track["track_sha256"].(string)
			if !ok || hash == "" {
				return nil, fmt.Errorf("release response has a missing exact track hash")
			}
			trackHashes = append(trackHashes, hash)
		}
		body := map[string]any{"review_id": ctx.Flag("review-id"), "revision": revision, "decision": ctx.Flag("decision"), "heard_all_tracks": true, "reviewed_track_sha256s": trackHashes, "note": ctx.Flag("note")}
		return request(core, "POST", "/releases/"+url.PathEscape(ctx.Positional("release_id"))+"/reviews", nil, body)
	}, func(_ cliapp.OperationContext, result map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Appended operator release review:", pretty(result)}}
	})

	rights := cliapp.Action(func(ctx cliapp.OperationContext) (map[string]any, error) {
		var revision int
		if _, err := fmt.Sscan(ctx.Flag("revision"), &revision); err != nil || revision < 1 {
			return nil, fmt.Errorf("revision must be a positive integer")
		}
		body := map[string]any{
			"decision_id": ctx.Flag("decision-id"), "revision": revision,
			"track_sha256": ctx.Flag("track-hash"), "source_license": ctx.Flag("source-license"),
			"usage_scope": ctx.Flag("usage-scope"), "decision": ctx.Flag("decision"),
			"evidence_ref": ctx.Flag("evidence-ref"), "note": ctx.Flag("note"),
		}
		return request(core, "POST", "/releases/"+url.PathEscape(ctx.Positional("release_id"))+"/rights-decisions", nil, body)
	}, func(_ cliapp.OperationContext, result map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Appended exact-hash operator rights decision:", pretty(result)}}
	})

	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"release-list":   list,
		"release-create": create,
		"release-get":    get,
		"release-rights": rights,
		"release-review": review,
	})
}

func pretty(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func request(core *cliapp.ScenarioApp, method, path string, query url.Values, body any) (map[string]any, error) {
	data, err := core.Request(method, path, query, body)
	if err != nil {
		return nil, cliapp.WrapAPIError("release "+method, err, data)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode Music Library release response: %w", err)
	}
	return result, nil
}
