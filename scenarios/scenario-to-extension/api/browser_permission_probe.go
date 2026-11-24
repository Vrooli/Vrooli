package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// browserPermissionProbePath identifies the artifact-owned page through which
// a managed BAS session can ask an extension to report its own current grants.
// It uses chrome.permissions.getAll() and does not add a manifest permission.
const browserPermissionProbePath = "__vrooli_bas/permission-report.html"

const browserPermissionProbeHTML = `<!doctype html>
<meta charset="utf-8">
<title>Vrooli extension permission report</title>
<pre id="report" data-status="pending">pending</pre>
<script src="permission-report.js"></script>
`

const browserPermissionProbeJS = `(() => {
  const output = document.getElementById('report');
  const normalize = (values) => [...new Set((Array.isArray(values) ? values : [])
    .filter((value) => typeof value === 'string')
    .map((value) => value.trim())
    .filter(Boolean))].sort();
  const publish = (status, detail) => {
    output.dataset.status = status;
    output.textContent = JSON.stringify(detail);
  };
  if (!globalThis.chrome?.runtime?.id || !globalThis.chrome?.permissions?.getAll) {
    publish('unavailable', { protocol: 'vrooli.extension-permissions.v1', reason: 'permissions_api_unavailable' });
    return;
  }
  Promise.resolve(chrome.permissions.getAll()).then((grants) => {
    const manifest = chrome.runtime.getManifest();
    publish('ready', {
      protocol: 'vrooli.extension-permissions.v1',
      extension_id: chrome.runtime.id,
      version: manifest.version,
      permissions: normalize(grants?.permissions),
      host_permissions: normalize(grants?.origins),
      observed_at: new Date().toISOString(),
      source: 'chrome.permissions.getAll'
    });
  }).catch(() => publish('unavailable', {
    protocol: 'vrooli.extension-permissions.v1', reason: 'permissions_api_failed'
  }));
})();
`

// ensureBrowserPermissionProbe adds a small same-extension report page to the
// release artifact. The source is deterministic and contains no manifest
// changes, network calls, permission requests, or extra host/API privileges.
func ensureBrowserPermissionProbe(root string) error {
	directory := filepath.Join(root, "__vrooli_bas")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create BAS permission-report directory: %w", err)
	}
	for name, want := range map[string][]byte{
		"permission-report.html": []byte(browserPermissionProbeHTML),
		"permission-report.js":   []byte(browserPermissionProbeJS),
	} {
		path := filepath.Join(directory, name)
		got, err := os.ReadFile(path)
		if err == nil {
			if !bytes.Equal(got, want) {
				return fmt.Errorf("reserved BAS permission-report asset already exists with different content: %s", name)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("read BAS permission-report asset %s: %w", name, err)
		}
		if err := os.WriteFile(path, want, 0o644); err != nil {
			return fmt.Errorf("write BAS permission-report asset %s: %w", name, err)
		}
	}
	return nil
}
