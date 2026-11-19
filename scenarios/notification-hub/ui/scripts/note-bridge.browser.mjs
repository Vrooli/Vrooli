import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { readFileSync, mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiDir = resolve(fileURLToPath(new URL("..", import.meta.url)));
const bridgePath = join(uiDir, "src/features/packet/explainer/frameBridge.js");
const chrome = ["google-chrome", "chromium", "chromium-browser"].find((candidate) => spawnSync(candidate, ["--version"], { stdio: "ignore" }).status === 0);

test("sandboxed browser note reaches the Hub answer payload", { skip: !chrome && "Chrome is not installed" }, () => {
  const temp = mkdtempSync(join(tmpdir(), "hub-note-bridge-"));
  try {
    const bridge = readFileSync(bridgePath, "utf8");
    const html = `<!doctype html><meta charset="utf-8"><pre id="result">waiting</pre>
      <textarea id="note">The output needs context.</textarea>
      <iframe id="page" sandbox="allow-scripts" srcdoc="${escapeAttribute(`<script>${bridge}<\/script><script>window.addEventListener('load',async()=>{const image=await hub.asset('evidence/image.png');if(image)hub.note('evidence.alpha','Evidence A')})<\/script>`)}"></iframe>
      <script>
        const frame = document.getElementById('page');
        window.addEventListener('message', event => {
          const message = event.data;
          if (event.source !== frame.contentWindow || message?.hub !== 1) return;
          if (message.type === 'asset') {
            frame.contentWindow.postMessage({ hub: 1, type: 'asset-resolve', requestId: message.requestId, dataUrl: 'data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=' }, '*');
            return;
          }
          if (message.type !== 'note') return;
          const changes = { notes: { [message.key]: { label: message.label, text: document.getElementById('note').value, attachments: [] } } };
          document.getElementById('result').textContent = JSON.stringify({ changes });
        });
      </script>`;
    const file = join(temp, "note.html");
    writeFileSync(file, html);
    const output = execFileSync(chrome, ["--headless", "--no-sandbox", "--disable-gpu", "--virtual-time-budget=1500", "--dump-dom", `file://${file}`], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] });
    const found = output.match(/<pre id="result">(.*?)<\/pre>/s);
    assert.ok(found, "browser rendered the result payload");
    assert.deepEqual(JSON.parse(found[1]), { changes: { notes: { "evidence.alpha": { label: "Evidence A", text: "The output needs context.", attachments: [] } } } });
  } finally {
    rmSync(temp, { recursive: true, force: true });
  }
});

function escapeAttribute(value) {
  return value.replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("<", "&lt;");
}
