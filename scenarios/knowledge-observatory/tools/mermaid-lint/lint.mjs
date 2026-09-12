import { JSDOM } from "jsdom";

function parserLine(error) {
  const match = String(error).match(/(?:on|at) line\s+(\d+)/i);
  return match ? Number(match[1]) : null;
}

function infrastructureFailure(message) {
  process.stderr.write(`mermaid-lint: ${message}\n`);
  process.exitCode = 1;
}

let input;
try {
  input = JSON.parse(await new Response(process.stdin).text());
} catch (error) {
  infrastructureFailure(`invalid JSON input: ${error.message}`);
  process.exit();
}

if (!Array.isArray(input.blocks)) {
  infrastructureFailure("input must contain a blocks array");
  process.exit();
}

const dom = new JSDOM("<!doctype html><html><body></body></html>", { url: "http://localhost" });
Object.assign(globalThis, {
  window: dom.window,
  document: dom.window.document,
  navigator: dom.window.navigator,
  HTMLElement: dom.window.HTMLElement,
  SVGElement: dom.window.SVGElement,
  getComputedStyle: dom.window.getComputedStyle,
});
const { default: mermaid } = await import("mermaid");
mermaid.initialize({ startOnLoad: false, securityLevel: "strict" });

const results = [];
for (const block of input.blocks) {
  const id = String(block?.id ?? "");
  const content = String(block?.content ?? "");
  try {
    await mermaid.parse(content);
    results.push({ id, valid: true, error: null, line: null });
  } catch (error) {
    results.push({ id, valid: false, error: String(error.message ?? error), line: parserLine(error) });
  }
}

process.stdout.write(`${JSON.stringify({ engine: "mermaid@11.13.0", results })}\n`);
