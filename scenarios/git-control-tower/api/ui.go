package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const fallbackHomePageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Git Control Tower</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 0; padding: 48px; background: #0d1117; color: #f0f6fc; }
    main { max-width: 720px; margin: 0 auto; padding: 32px; border-radius: 12px; background: rgba(13, 17, 23, 0.92); border: 1px solid rgba(240, 246, 252, 0.08); }
    h1 { margin-top: 0; }
    p { line-height: 1.6; }
    code { background: rgba(240, 246, 252, 0.08); padding: 2px 6px; border-radius: 4px; }
  </style>
</head>
<body>
  <main>
    <h1>Git Control Tower</h1>
    <p>The interactive UI assets are unavailable. Ensure <code>scenarios/git-control-tower/ui/index.html</code> exists and reload the page.</p>
    <p>If you just installed the scenario, run <code>make start</code> from the scenario directory to rebuild resources.</p>
  </main>
</body>
</html>`

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	if dir := resolveStaticUIDir(); dir != "" {
		indexPath := filepath.Join(dir, "index.html")
		if data, err := os.ReadFile(indexPath); err == nil {
			_, _ = w.Write(data)
			return
		} else if !os.IsNotExist(err) {
			log.Printf("git-control-tower: unable to read UI index: %v", err)
		}
	}

	_, _ = w.Write([]byte(fallbackHomePageHTML))
}
