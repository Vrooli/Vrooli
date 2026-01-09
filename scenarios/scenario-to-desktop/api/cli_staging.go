package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// stageCLIs copies CLI binaries into bundle/bin and writes a thin vrooli shim for control API access.
func stageCLIs(bundleRoot, platform string) error {
	binDir := filepath.Join(bundleRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	cliDir := filepath.Join(bundleRoot, "cli")
	if info, err := os.Stat(cliDir); err == nil && info.IsDir() {
		if err := filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			dst := filepath.Join(binDir, filepath.Base(path))
			if err := copyPath(path, dst); err != nil {
				return err
			}
			return os.Chmod(dst, 0o755)
		}); err != nil {
			return err
		}
	}

	// Only create the bash shim for non-Windows targets.
	if strings.HasPrefix(platform, "win") {
		return nil
	}

	shim := filepath.Join(binDir, "vrooli")
	script := fmt.Sprintf(vrooliShimTemplate, platform)
	return os.WriteFile(shim, []byte(script), 0o755)
}

const vrooliShimTemplate = `#!/usr/bin/env bash
set -euo pipefail

bundle_root="$(cd "$(dirname "$0")"/.. && pwd)"
manifest_path="$bundle_root/bundle.json"
runtime_ctl="$bundle_root/runtime/%s/runtimectl"

if [ ! -x "$runtime_ctl" ]; then
  echo "runtimectl not found at $runtime_ctl" >&2
  exit 1
fi

app_name=$(python3 - "$manifest_path" <<'PY'
import json, re, sys
m=json.load(open(sys.argv[1]))
name=m.get("app", {}).get("name", "app")
safe=re.sub(r'[^A-Za-z0-9]+', '-', name).strip('-').lower() or "app"
print(safe)
PY
)
config_root="${XDG_CONFIG_HOME:-$HOME/.config}"
token_file="$config_root/${app_name}/runtime/auth_token"
port_file="$config_root/${app_name}/runtime/ipc_port"
ipc_port=$(python3 - "$manifest_path" <<'PY'
import json, sys
m=json.load(open(sys.argv[1]))
print(m.get("ipc", {}).get("port", 39200))
PY
)
if [ -f "$port_file" ]; then
  maybe_port=$(cat "$port_file" | tr -d '[:space:]')
  if [[ "$maybe_port" =~ ^[0-9]+$ ]]; then
    ipc_port="$maybe_port"
  fi
fi

cmd="${1:-}"
shift || true

if [ "$cmd" = "scenario" ] && [ "${1:-}" = "port" ]; then
  scenario="${2:-}"
  port_var="${3:-}"
  if [ -z "$port_var" ]; then
    echo "usage: vrooli scenario port <scenario> <API_PORT|UI_PORT>" >&2
    exit 1
  fi

  svcType=""
  case "$port_var" in
    API_PORT) svcType="api-binary" ;;
    UI_PORT) svcType="ui-bundle" ;;
    *)
      echo "unsupported port name: $port_var" >&2
      exit 1
      ;;
  esac

  read -r svcId portName <<'EOF' || true
$(python3 - "$manifest_path" "$svcType" <<'PY'
import json, sys
m=json.load(open(sys.argv[1])); typ=sys.argv[2]
for svc in m.get("services", []):
    if svc.get("type") == typ:
        svc_id = svc.get("id")
        ports = (svc.get("ports") or {}).get("requested") or []
        port_name = ports[0].get("name") if ports else "http"
        print(f"{svc_id} {port_name}")
        sys.exit(0)
print("")
PY
)
EOF

  if [ -z "${svcId:-}" ]; then
    echo "no service for type $svcType" >&2
    exit 1
  fi

  exec "$runtime_ctl" --port "$ipc_port" --token-file "$token_file" port --service "$svcId" --port-name "$portName"
fi

# Fallback: pass through to runtimectl
exec "$runtime_ctl" --port "$ipc_port" --token-file "$token_file" "$cmd" "$@"
`
