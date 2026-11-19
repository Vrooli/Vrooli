import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

import { describe, expect, it } from "vitest";

import { BOUNDARY_FACTS_VERSION } from "../src/boundary-facts.js";
import { extractBoundaryFactsRequest } from "../src/boundary-facts-cli.js";

describe("boundary facts CLI request", () => {
  it("parses only caller-selected source files and returns a versioned deterministic response", () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "tcg-boundary-cli-"));
    try {
      fs.mkdirSync(path.join(root, "ui"), { recursive: true });
      fs.writeFileSync(path.join(root, "ui", "peer.ts"), "fetch('/embedded/swarm-manager/api/v1/backlog');\n");
      fs.writeFileSync(path.join(root, "ui", "ignored.ts"), "fetch('/embedded/unselected-scenario/api');\n");

      const request = { protocol_version: BOUNDARY_FACTS_VERSION, root_dir: root, files: ["ui/peer.ts"] };
      const first = extractBoundaryFactsRequest(request);
      const second = extractBoundaryFactsRequest(request);

      expect(first).toEqual(second);
      expect(first.protocol_version).toBe(BOUNDARY_FACTS_VERSION);
      expect(first.facts).toHaveLength(1);
      expect(first.facts[0]?.target).toEqual({ kind: "scenario", name: "swarm-manager" });
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  it("rejects unsupported versions and paths outside the repository root", () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "tcg-boundary-cli-invalid-"));
    try {
      fs.writeFileSync(path.join(root, "peer.ts"), "fetch('/embedded/swarm-manager/api');\n");

      expect(() => extractBoundaryFactsRequest({ protocol_version: 2, root_dir: root, files: [] }))
        .toThrow(/unsupported boundary facts protocol/);
      expect(() => extractBoundaryFactsRequest({ protocol_version: BOUNDARY_FACTS_VERSION, root_dir: root, files: ["../escape.ts"] }))
        .toThrow(/escapes root_dir/);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
