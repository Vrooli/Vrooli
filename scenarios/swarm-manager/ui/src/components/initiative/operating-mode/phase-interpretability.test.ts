import { describe, expect, it } from "vitest";
import { PHASE_RESULT_FIELDS } from "./phase-interpretability";

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
  // In v2 the Reads tab derives from each mode's declared, composed contract
  // (base ∪ target adapter) — there is no fixed reads vocabulary to keep in
  // lockstep with the backend, so that alignment test is retired. The emits
  // schema is still a fixed projection of the PhaseResult contract, so it
  // remains pinned here.
  it("keeps the emits schema aligned with PhaseResult json fields", () => {
    const outputGo = readText(path.resolve?.(scenarioRoot, "api/internal/operatingmode/output.go") ?? "");

    for (const field of PHASE_RESULT_FIELDS) {
      expect(outputGo).toContain(`json:"${field}`);
    }
  });
});
