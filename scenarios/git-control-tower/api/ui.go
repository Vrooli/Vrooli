package main

import "net/http"

const homePageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Git Control Tower</title>
  <style>
    :root {
      color-scheme: dark light;
      font-family: "Inter", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }

    body {
      margin: 0;
      padding: 0;
      background: #0d1117;
      color: #f0f6fc;
      min-height: 100vh;
      display: flex;
      justify-content: center;
      align-items: flex-start;
    }

    .app {
      width: min(960px, 90vw);
      margin: 48px auto;
      background: rgba(13, 17, 23, 0.92);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(240, 246, 252, 0.08);
      border-radius: 16px;
      padding: 32px 36px 40px;
      box-shadow: 0 18px 48px rgba(1, 4, 9, 0.45);
    }

    header h1 {
      font-size: clamp(2rem, 4vw, 2.6rem);
      margin: 0;
      letter-spacing: -0.02em;
    }

    header p {
      margin: 12px 0 0;
      color: rgba(240, 246, 252, 0.7);
      line-height: 1.5;
    }

    .grid {
      display: grid;
      gap: 16px;
      margin-top: 28px;
    }

    .metrics {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 16px;
    }

    .metric-box {
      border-radius: 12px;
      padding: 16px 18px;
      background: rgba(22, 27, 34, 0.85);
      border: 1px solid rgba(240, 246, 252, 0.06);
    }

    .metric-box span {
      display: block;
      text-transform: uppercase;
      font-size: 12px;
      letter-spacing: 0.12em;
      color: rgba(240, 246, 252, 0.55);
    }

    .metric-box strong {
      display: block;
      margin-top: 8px;
      font-size: 1.35rem;
      font-weight: 600;
    }

    ul.file-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 10px;
    }

    ul.file-list li {
      padding: 12px 14px;
      border-radius: 10px;
      background: rgba(22, 27, 34, 0.85);
      border: 1px solid rgba(240, 246, 252, 0.05);
      display: grid;
      gap: 2px;
    }

    ul.file-list li span.path {
      font-family: "JetBrains Mono", Menlo, Consolas, monospace;
      font-size: 0.95rem;
      word-break: break-word;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      padding: 4px 10px;
      border-radius: 999px;
      background: rgba(88, 166, 255, 0.18);
      color: #58a6ff;
      border: 1px solid rgba(88, 166, 255, 0.35);
      margin-top: 6px;
    }

    .badge.scope {
      background: rgba(163, 113, 247, 0.2);
      color: #c995ff;
      border-color: rgba(163, 113, 247, 0.35);
    }

    .panel {
      border-radius: 14px;
      padding: 20px 22px;
      background: rgba(13, 17, 23, 0.85);
      border: 1px solid rgba(240, 246, 252, 0.08);
    }

    .panel h2 {
      margin: 0 0 12px;
      font-size: 1.2rem;
    }

    .status-message {
      margin-top: 18px;
      border-radius: 12px;
      padding: 16px 18px;
      background: rgba(248, 81, 73, 0.16);
      border: 1px solid rgba(248, 81, 73, 0.32);
      color: #ffaba6;
    }

    .status-message.success {
      background: rgba(35, 134, 54, 0.18);
      border-color: rgba(46, 160, 67, 0.32);
      color: #7ee787;
    }

    footer {
      margin-top: 32px;
      font-size: 0.85rem;
      color: rgba(240, 246, 252, 0.55);
    }

    a {
      color: #58a6ff;
    }

    @media (max-width: 680px) {
      .app {
        width: 92vw;
        padding: 24px 22px 32px;
        margin: 32px auto;
      }

      .metrics {
        grid-template-columns: 1fr;
      }
    }
  </style>
</head>
<body>
  <main class="app">
    <header>
      <h1>Git Control Tower</h1>
      <p>Monitor repository health, review staged changes, and keep your git workflow agent-friendly. The dashboard pulls live data from the Git Control Tower API.</p>
    </header>

    <section id="health" class="panel">
      <h2>Service Health</h2>
      <div id="health-status">Checking API health…</div>
    </section>

    <section id="repo" class="panel">
      <h2>Repository Snapshot</h2>
      <div id="repo-content">Loading repository status…</div>
    </section>

    <footer>
      API base URL: <code>/api/v1/</code> • CLI: <code>git-control-tower &lt;command&gt;</code>
    </footer>
  </main>

  <script>
    async function fetchJSON(url) {
      const response = await fetch(url, { headers: { "Accept": "application/json" } });
      if (!response.ok) {
        throw new Error("Request failed with status " + response.status);
      }
      return response.json();
    }

    function renderHealth(data) {
      const container = document.getElementById("health-status");
      const readiness = data.readiness ? "Operational" : "Degraded";
      const statusClass = data.readiness ? "status-message success" : "status-message";

      const dependencies = Object.entries(data.dependencies || {}).map(([name, details]) => {
        if (details === null || typeof details !== "object") {
          return "";
        }
        const pretty = name.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
        const detailStatus = Object.entries(details).map(([key, value]) => key + ": " + value).join(" • ");
        return '<li><span class="path">' + pretty + '</span><span class="badge">' + detailStatus + '</span></li>';
      }).join("");

      container.innerHTML =
        '<div class="' + statusClass + '"><strong>' + readiness + '</strong><br/>Last checked: ' + new Date(data.timestamp).toLocaleString() + '</div>' +
        (dependencies ? '<h3 style="margin-top:18px;">Dependencies</h3><ul class="file-list">' + dependencies + '</ul>' : '');
    }

    function renderRepository(status) {
      const container = document.getElementById("repo-content");
      if (!status || typeof status !== 'object') {
        container.innerHTML = '<div class="status-message">Repository data is unavailable.</div>';
        return;
      }
      const tracking = status && typeof status === 'object' && status.tracking ? status.tracking : {};
      const staged = Array.isArray(status.staged) ? status.staged : [];
      const unstaged = Array.isArray(status.unstaged) ? status.unstaged : [];
      const untracked = Array.isArray(status.untracked) ? status.untracked : [];
      const conflicts = Array.isArray(status.conflicts) ? status.conflicts : [];

      const metrics =
        '<div class="metrics">' +
          '<div class="metric-box"><span>Branch</span><strong>' + (status.branch || 'unknown') + '</strong></div>' +
          '<div class="metric-box"><span>Tracking</span><strong>' + (tracking.branch || "n/a") + '</strong><small style="display:block;margin-top:6px;color:rgba(240,246,252,0.6);">Ahead ' + (tracking.ahead || 0) + ' • Behind ' + (tracking.behind || 0) + '</small></div>' +
          '<div class="metric-box"><span>Staged Files</span><strong>' + staged.length + '</strong></div>' +
          '<div class="metric-box"><span>Unstaged Files</span><strong>' + unstaged.length + '</strong></div>' +
        '</div>';

      const stagedList = staged.map((file) => {
        const scopeBadge = file.scope ? '<span class="badge scope">' + file.scope + '</span>' : "";
        return '<li><span class="path">' + file.path + '</span><span class="badge">' + file.status + '</span>' + scopeBadge + '</li>';
      }).join("") || '<p class="status-message">No staged changes found.</p>';

      const unstagedList = unstaged.map((file) => {
        const scopeBadge = file.scope ? '<span class="badge scope">' + file.scope + '</span>' : "";
        return '<li><span class="path">' + file.path + '</span><span class="badge">' + file.status + '</span>' + scopeBadge + '</li>';
      }).join("") || '<p class="status-message">No unstaged changes found.</p>';

      const untrackedList = untracked.map((file) => '<li><span class="path">' + file + '</span></li>').join("") || '<p class="status-message">No untracked files detected.</p>';

      const conflictList = conflicts.map((file) => '<li><span class="path">' + file + '</span></li>').join("");
      const conflictsSection = conflictList ? '<h3>Merge Conflicts</h3><ul class="file-list">' + conflictList + '</ul>' : '';

      container.innerHTML =
        metrics +
        '<div class="grid" style="margin-top:28px;">' +
          '<section><h3>Staged Changes</h3><ul class="file-list">' + stagedList + '</ul></section>' +
          '<section><h3>Unstaged Changes</h3><ul class="file-list">' + unstagedList + '</ul></section>' +
          '<section><h3>Untracked Files</h3><ul class="file-list">' + untrackedList + '</ul></section>' +
          conflictsSection +
        '</div>';
    }

    async function bootstrap() {
      try {
        const [health, status] = await Promise.all([
          fetchJSON('/health'),
          fetchJSON('/api/v1/status')
        ]);
        renderHealth(health);
        renderRepository(status);
      } catch (error) {
        document.getElementById('health-status').innerHTML = '<div class="status-message">' + error.message + '</div>';
        document.getElementById('repo-content').innerHTML = '<div class="status-message">Unable to load repository data. Check API logs for details.</div>';
      }
    }

    bootstrap();
  </script>
</body>
</html>`

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(homePageHTML))
}
