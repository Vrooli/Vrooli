package permissions

// One-time migration from the retired in-config intent record. Earlier
// adapters kept Vrooli's intent in a top-level `[vrooli]` table inside
// config.toml. Codex never read it and, from 0.166, warns about it on every
// launch ("`vrooli` is ignored"). The table is read only to move it into the
// intent sidecar: `resource-codex ensure` (run by `vrooli setup`) or the next
// permissions write strips it. Delete this file once no supported
// installation can still carry the table.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
	"github.com/vrooli/agentharness"
)

const legacyTableKey = "vrooli"

// legacyIntent decodes the retired table; ok reports whether it is present.
func legacyIntent(doc map[string]any) (Intent, bool, error) {
	raw, ok := doc[legacyTableKey]
	if !ok {
		return Intent{}, false, nil
	}
	section, _ := raw.(map[string]any)
	perms, _ := section["permissions"].(map[string]any)
	intent := Intent{BashDeny: decodeStringArray(perms, "bash_deny"), BashAsk: decodeStringArray(perms, "bash_ask"), BashAllow: decodeStringArray(perms, "bash_allow")}
	if rawExecution, ok := section["execution"]; ok {
		encoded, err := json.Marshal(rawExecution)
		if err != nil {
			return Intent{}, true, err
		}
		if err = json.Unmarshal(encoded, &intent.Execution); err != nil {
			return Intent{}, true, err
		}
		if err = agentharness.ValidateExecutionPermissions(intent.Execution); err != nil {
			return Intent{}, true, fmt.Errorf("invalid execution intent in the legacy [vrooli] table: %w", err)
		}
		intent.ExecutionActive, _ = section["execution_active"].(bool)
	}
	return intent, true, nil
}

func decodeStringArray(m map[string]any, key string) []string {
	arr, _ := m[key].([]any)
	if arr == nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// stripLegacyTable removes every `vrooli` expression from config bytes and
// keeps all other bytes, including comments and key order, so the migration
// does not reformat a file the user may never have asked Vrooli to rewrite.
// The result must decode to the original document minus the table.
func stripLegacyTable(data []byte) ([]byte, error) {
	var want map[string]any
	if err := toml.Unmarshal(data, &want); err != nil {
		return nil, err
	}
	if _, ok := want[legacyTableKey]; !ok {
		return data, nil
	}
	delete(want, legacyTableKey)
	out, err := cutLegacyExpressions(data)
	if err != nil {
		return nil, err
	}
	var got map[string]any
	if err = toml.Unmarshal(out, &got); err != nil || !reflect.DeepEqual(got, want) {
		return nil, fmt.Errorf("could not remove the [%s] table without changing other settings (%v); remove it by hand", legacyTableKey, err)
	}
	return out, nil
}

// cutLegacyExpressions drops table headers whose first key part is `vrooli`
// with their key/values, and root key/values whose first key part is
// `vrooli`. Comments between two dropped expressions go with them; comments
// after the last one belong to what follows and stay.
func cutLegacyExpressions(data []byte) ([]byte, error) {
	type span struct{ start, end int }
	var dropped []span
	var parser unstable.Parser
	parser.Reset(data)
	inTable, inLegacy := false, false
	for parser.NextExpression() {
		expr := parser.Expression()
		if expr.Kind != unstable.Table && expr.Kind != unstable.ArrayTable && expr.Kind != unstable.KeyValue {
			continue
		}
		keys := expr.Key()
		keys.Next()
		first := keys.Node()
		start, end := int(first.Raw.Offset), int(first.Raw.Offset+first.Raw.Length)
		drop := inLegacy
		if expr.Kind == unstable.KeyValue {
			end = int(expr.Raw.Offset + expr.Raw.Length)
			drop = drop || (!inTable && string(first.Data) == legacyTableKey)
		} else {
			inTable = true
			inLegacy = string(first.Data) == legacyTableKey
			drop = inLegacy
		}
		if !drop {
			continue
		}
		s := span{start: lineStart(data, start), end: lineEnd(data, end)}
		if n := len(dropped); n > 0 && len(bytes.TrimSpace(stripComments(data[dropped[n-1].end:s.start]))) == 0 {
			dropped[n-1].end = s.end
			continue
		}
		dropped = append(dropped, s)
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	cursor := 0
	for _, s := range dropped {
		// Do not leave a doubled blank line, or a trailing one at the end.
		if s.start >= 2 && data[s.start-1] == '\n' && data[s.start-2] == '\n' {
			if s.end == len(data) {
				s.start--
			} else if data[s.end] == '\n' {
				s.end++
			}
		}
		out.Write(data[cursor:s.start])
		cursor = s.end
	}
	out.Write(data[cursor:])
	return out.Bytes(), nil
}

func lineStart(data []byte, offset int) int {
	return bytes.LastIndexByte(data[:offset], '\n') + 1
}

// lineEnd returns the offset just past the newline that ends offset's line.
func lineEnd(data []byte, offset int) int {
	if i := bytes.IndexByte(data[offset:], '\n'); i >= 0 {
		return offset + i + 1
	}
	return len(data)
}

// stripComments blanks `#` comments line by line; the input holds only
// whitespace and comments, never values, so no string can contain `#`.
func stripComments(between []byte) []byte {
	var out []byte
	for _, line := range bytes.Split(between, []byte("\n")) {
		if i := bytes.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		out = append(out, line...)
	}
	return out
}
