package desktopcomponent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectServices is the one service projection for an embedded dependency and
// a later installed copy. It preserves unknown declarative fields rather than
// silently narrowing the service contract at a consumer's typed boundary.
func ProjectServices(owner string, raw json.RawMessage) (json.RawMessage, error) {
	if !ValidID(owner) {
		return nil, fmt.Errorf("invalid component owner")
	}
	var services []map[string]any
	if err := json.Unmarshal(raw, &services); err != nil {
		return nil, err
	}
	ids := map[string]string{}
	prefix := owner + "--"
	for _, svc := range services {
		id, _ := svc["id"].(string)
		if !ValidID(id) {
			return nil, fmt.Errorf("invalid component service ID")
		}
		if strings.HasPrefix(id, prefix) {
			ids[id] = id
		} else {
			ids[id] = prefix + id
		}
	}
	sourcePath := func(value any) (any, error) {
		text, _ := value.(string)
		if text == "" {
			return value, nil
		}
		if !filepath.IsLocal(filepath.FromSlash(text)) {
			return nil, fmt.Errorf("component path must be local")
		}
		return filepath.ToSlash(filepath.Join("peers", owner, filepath.FromSlash(text))), nil
	}
	for _, svc := range services {
		old := svc["id"].(string)
		if old == ids[old] {
			continue
		}
		svc["id"] = ids[old]
		if dependencies, ok := svc["dependencies"].([]any); ok {
			for i, dep := range dependencies {
				if id, ok := dep.(string); ok {
					mapped, found := ids[id]
					if !found {
						return nil, fmt.Errorf("component dependency %s is outside its service graph", id)
					}
					dependencies[i] = mapped
				}
			}
		}
		if values, ok := svc["env"].(map[string]any); ok {
			for key, value := range values {
				if text, ok := value.(string); ok {
					for id, mapped := range ids {
						text = strings.ReplaceAll(text, "${"+id+".", "${"+mapped+".")
					}
					values[key] = text
				}
			}
		}
		if build, ok := svc["build"].(map[string]any); ok {
			value, err := sourcePath(build["source_dir"])
			if err != nil {
				return nil, err
			}
			build["source_dir"] = value
			if output, ok := build["output_pattern"].(string); ok {
				build["output_pattern"] = strings.ReplaceAll(output, "/"+old+"{{ext}}", "/"+ids[old]+"{{ext}}")
			}
		}
		if bins, ok := svc["binaries"].(map[string]any); ok {
			for _, value := range bins {
				if bin, ok := value.(map[string]any); ok {
					v, err := sourcePath(bin["path"])
					if err != nil {
						return nil, err
					}
					bin["path"] = v
					if cwd, ok := bin["cwd"].(string); ok && cwd != "" {
						v, err = sourcePath(cwd)
						if err != nil {
							return nil, err
						}
						bin["cwd"] = v
					}
				}
			}
		}
		if assets, ok := svc["assets"].([]any); ok {
			for _, v := range assets {
				if asset, ok := v.(map[string]any); ok {
					value, err := sourcePath(asset["path"])
					if err != nil {
						return nil, err
					}
					asset["path"] = value
				}
			}
		}
		if dirs, ok := svc["asset_dirs"].([]any); ok {
			for i, v := range dirs {
				value, err := sourcePath(v)
				if err != nil {
					return nil, err
				}
				dirs[i] = value
			}
		}
		if text, ok := svc["dist_root"].(string); ok && text != "" {
			value, err := sourcePath(text)
			if err != nil {
				return nil, err
			}
			svc["dist_root"] = value
		}
	}
	return json.Marshal(services)
}
