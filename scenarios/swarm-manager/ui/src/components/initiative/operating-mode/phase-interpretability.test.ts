import { describe, expect, it } from "vitest";
import { PHASE_READS, PHASE_RESULT_FIELDS } from "./phase-interpretability";

declare const require: (id: string) => {
  readFileSync?: (path: string, encoding: string) => string;
  resolve?: (...paths: string[]) => string;
  cwd?: () => string;
};

const fs = require("fs");
const path = require("path");
const processShim = require("process");
const scenarioRoot = path.resolve?.(processShim.cwd?.() ?? ".", "..") ?? "..";

function readText(pathname: string): string {
  const readFileSync = fs.readFileSync;
  if (!readFileSync) throw new Error("fs.readFileSync unavailable");
  return readFileSync(pathname, "utf8");
}

describe("phase interpretability config", () => {
  it("keeps the reads list aligned with prompt variables", () => {
    const promptGo = readText(path.resolve?.(scenarioRoot, "api/internal/operatingmode/prompt.go") ?? "");

    for (const read of PHASE_READS) {
      expect(promptGo).toContain(`"${read.key}"`);
    }
  });

  it("keeps the emits schema aligned with PhaseResult json fields", () => {
    const outputGo = readText(path.resolve?.(scenarioRoot, "api/internal/operatingmode/output.go") ?? "");

    for (const field of PHASE_RESULT_FIELDS) {
      expect(outputGo).toContain(`json:"${field}`);
    }
  });
});
