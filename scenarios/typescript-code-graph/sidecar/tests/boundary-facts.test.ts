import { Project } from "ts-morph";
import { describe, expect, it } from "vitest";

import { BOUNDARY_FACTS_VERSION, extractBoundaryFacts } from "../src/boundary-facts.js";

describe("boundary facts", () => {
  it("emits deterministic, located facts for embedded fetch calls and explicit unknown targets", () => {
    const project = new Project({ useInMemoryFileSystem: true, compilerOptions: { jsx: 2 } });
    project.createSourceFile("/repo/scenarios/prompt-manager/ui/src/board.tsx", `
// fetch('/embedded/commented-peer/api')
const prose = "fetch('/embedded/string-peer/api')";
const pattern = /fetch\\('\/embedded\\/regex-peer/;
const unrelated = '/embedded/literal-peer/path';
const same = fetch('/embedded/prompt-manager/api/v1/local');
const peer = fetch('/embedded/swarm-manager/api/v1/backlog?archived=false');
const jsx = <div title="/embedded/prop-peer/api">/embedded/text-peer/api</div>;
`);
    project.createSourceFile("/repo/scenarios/deployment-manager/ui/src/analyze.tsx", `
fetch("/embedded/scenario-dependency-analyzer/external-url");
fetch("/embedded/scenario-dependency-analyzer");
fetch("/embedded");
fetch(\`/embedded/\${scenarioName}/path\`);
fetch('/embedded/' + scenarioName + '/path');
fetch(\`/embedded/\${encodeURIComponent('agent-manager')}/api/v1/runs/\${runId}\`);
fetch(\`/embedded/\${encodeURIComponent(targetName)}/api\`);
`);
    const actual = extractBoundaryFacts({
      project,
      rootDir: "/repo",
      files: [
        "scenarios/prompt-manager/ui/src/board.tsx",
        "scenarios/deployment-manager/ui/src/analyze.tsx",
      ],
    });

    expect(actual).toEqual([
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 2, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "scenario", name: "scenario-dependency-analyzer" },
        transport: "rest",
        addressing: "embedded_path",
        evidence: { syntax: "fetch_literal_url", text: '"/embedded/scenario-dependency-analyzer/external-url"' },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 3, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "scenario", name: "scenario-dependency-analyzer" },
        transport: "rest",
        addressing: "embedded_path",
        evidence: { syntax: "fetch_literal_url", text: '"/embedded/scenario-dependency-analyzer"' },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 5, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "unknown", name: null },
        transport: "rest",
        addressing: "dynamic_embedded_path",
        evidence: { syntax: "fetch_dynamic_embedded_url", text: "`/embedded/${scenarioName}/path`" },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 6, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "unknown", name: null },
        transport: "rest",
        addressing: "dynamic_embedded_path",
        evidence: { syntax: "fetch_dynamic_embedded_url", text: "'/embedded/' + scenarioName + '/path'" },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 7, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "scenario", name: "agent-manager" },
        transport: "rest",
        addressing: "embedded_path",
        evidence: { syntax: "fetch_dynamic_url_with_literal_target", text: "`/embedded/${encodeURIComponent('agent-manager')}/api/v1/runs/${runId}`" },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/deployment-manager/ui/src/analyze.tsx", line: 8, column: 1 },
        boundaryKind: "embedded_http_request",
        target: { kind: "unknown", name: null },
        transport: "rest",
        addressing: "dynamic_embedded_path",
        evidence: { syntax: "fetch_dynamic_embedded_url", text: "`/embedded/${encodeURIComponent(targetName)}/api`" },
      },
      {
        version: BOUNDARY_FACTS_VERSION,
        source: { path: "scenarios/prompt-manager/ui/src/board.tsx", line: 7, column: 14 },
        boundaryKind: "embedded_http_request",
        target: { kind: "scenario", name: "swarm-manager" },
        transport: "rest",
        addressing: "embedded_path",
        evidence: { syntax: "fetch_literal_url", text: "'/embedded/swarm-manager/api/v1/backlog?archived=false'" },
      },
    ]);
  });

  it("does not infer calls from arbitrary peer literals, declarations, or unsupported resolver shapes", () => {
    const project = new Project({ useInMemoryFileSystem: true });
    project.createSourceFile("/p/src/negative.ts", `
const url = '/embedded/peer/api';
function fetch(url: string) {}
fetch(url);
obj.fetch('/embedded/peer/api');
new fetch('/embedded/peer/api');
const resolveScenarioUrl = (name: string) => name;
const reference = resolveScenarioUrl;
function unrelatedResolveScenarioUrl(name: string) { return name; }
function test() {
  function resolveScenarioUrl(name: string) { return name; }
  resolveScenarioUrl('not-a-real-boundary');
  unrelated.resolveScenarioUrl('not-a-real-boundary');
}
`);
    expect(extractBoundaryFacts({ project, rootDir: "/p", files: ["src/negative.ts"] })).toEqual([]);
  });

  it("does not treat a lexically shadowed fetch as the HTTP global", () => {
    const project = new Project({ useInMemoryFileSystem: true });
    project.createSourceFile("/p/src/shadowed.ts", `
fetch('/embedded/real-peer/api');
function fetch(url: string) { return url; }
`);
    expect(extractBoundaryFacts({ project, rootDir: "/p", files: ["src/shadowed.ts"] })).toEqual([]);
  });

  it("does not classify loop, case, or function-scoped fetch bindings as the global HTTP fetch", () => {
    const project = new Project({ useInMemoryFileSystem: true });
    project.createSourceFile("/repo/scenarios/example/ui/src/loop-shadow.ts", `
for (const fetch of fakeFetchers) fetch('/embedded/not-a-peer/api');
switch (kind) {
  case 'local': {
    const { fetch } = fakeHandlers;
    fetch('/embedded/not-a-peer/api');
  }
  case 'function':
    function fetch(url: string) { return url; }
    fetch('/embedded/not-a-peer/api');
    break;
  case 'class':
    class fetch {}
    fetch('/embedded/not-a-peer/api');
    break;
}
switch (kind) {
  case 'module-var': {
    { var fetch = fakeFetch; }
    break;
  }
  case 'after-var':
    fetch('/embedded/not-a-peer/api');
    break;
}
`);
    project.createSourceFile("/repo/scenarios/example/ui/src/function-var-shadow.ts", `
function useLocalFetch() {
  if (condition) { var fetch = fakeFetch; }
  fetch('/embedded/not-a-peer/api');
}
`);
    project.createSourceFile("/repo/scenarios/example/ui/src/module-var-shadow.ts", `
if (condition) { { var fetch = fakeFetch; } }
fetch('/embedded/not-a-peer/api');
`);
    expect(extractBoundaryFacts({
      project,
      rootDir: "/repo",
      files: [
        "scenarios/example/ui/src/loop-shadow.ts",
        "scenarios/example/ui/src/function-var-shadow.ts",
        "scenarios/example/ui/src/module-var-shadow.ts",
      ],
    })).toEqual([]);
  });
});
