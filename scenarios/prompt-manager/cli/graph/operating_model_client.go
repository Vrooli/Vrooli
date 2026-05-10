package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"
	"strings"
)

func printRawOperatingModelJSON(ctx appctx.Context, path string, query url.Values) error {
	raw, err := rawOperatingModelJSON(ctx, path, query)
	if err != nil {
		return err
	}
	raw = formatRawOperatingModelJSON(raw)
	os.Stdout.Write(raw)
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func rawOperatingModelJSON(ctx appctx.Context, path string, query url.Values) ([]byte, error) {
	rawGetter, ok := ctx.(interface {
		GetRawWithQuery(string, url.Values) ([]byte, error)
	})
	if !ok {
		var fallback json.RawMessage
		if err := ctx.GetWithQuery(path, query, &fallback); err != nil {
			return nil, err
		}
		return fallback, nil
	}
	return rawGetter.GetRawWithQuery(path, query)
}

func formatRawOperatingModelJSON(raw []byte) []byte {
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return raw
	}
	return out.Bytes()
}

func operatingModelQuery(team, id string) url.Values {
	q := url.Values{}
	if strings.TrimSpace(team) != "" {
		q.Set("team", team)
	}
	if strings.TrimSpace(id) != "" {
		q.Set("id", id)
	}
	return q
}
