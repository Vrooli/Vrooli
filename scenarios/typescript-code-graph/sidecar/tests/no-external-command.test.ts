// REQ: the sidecar must NEVER spawn git, tsc, pnpm, or any child process
// against consumer projects. ESM module bindings are read-only, so we
// use vi.mock() to replace child_process before the modules under test
// load, then assert the mock fns are never called during extract+rewrite.

import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

import { Project } from "ts-morph";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const watchedFns = [
  "spawn",
  "spawnSync",
  "exec",
  "execSync",
  "execFile",
  "execFileSync",
  "fork",
] as const;

// Trackers shared with the mock factory via module-scoped state.
const calls: string[] = [];
const guardFactory = (name: string) =>
  vi.fn((...args: unknown[]) => {
    calls.push(`${name}(${JSON.stringify(args[0])})`);
    throw new Error(`forbidden child_process.${name} call`);
  });

vi.mock("node:child_process", () => {
  const obj: Record<string, unknown> = {};
  for (const fn of watchedFns) {
    obj[fn] = guardFactory(fn);
  }
  return obj;
});

// Import the modules under test AFTER vi.mock is hoisted.
const { extract } = await import("../src/extract.js");
const { applyRewrite } = await import("../src/rewrite.js");

beforeEach(() => {
  calls.length = 0;
});

afterEach(() => {
  // nothing to restore — the module mock persists across tests
});

describe("no external command", () => {
  it("extract on an in-memory project never spawns a child process", () => {
    const project = new Project({ useInMemoryFileSystem: true });
    project.createSourceFile(
      "/proj/src/a.ts",
      `/** doc */ export const A = 1;`,
    );
    const out = extract({
      scenarioPath: "/proj",
      _project: project,
      _rootDirOverride: "/proj",
    });
    expect(out.graph.nodes.length).toBeGreaterThan(0);
    expect(calls).toEqual([]);
  });

  it("rewrite on a temp project never spawns a child process", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "tcg-noext-"));
    try {
      fs.writeFileSync(
        path.join(tmp, "tsconfig.json"),
        JSON.stringify({
          compilerOptions: {
            target: "ES2022",
            module: "NodeNext",
            moduleResolution: "NodeNext",
            strict: true,
          },
          include: ["src/**/*"],
        }),
      );
      fs.mkdirSync(path.join(tmp, "src"));
      fs.writeFileSync(path.join(tmp, "src/util.ts"), `export const U = 1;\n`);
      fs.writeFileSync(
        path.join(tmp, "src/main.ts"),
        `import { U } from "./util";\nexport const M = U;\n`,
      );

      const results = await applyRewrite({
        scenarioPath: tmp,
        operations: [
          { import_rewrite: { old_path: "./util", new_path: "./renamed" } },
        ],
      });
      expect(results[0]!.ok).toBe(true);
      expect(calls).toEqual([]);
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  it("mock guard is effective (control)", async () => {
    // Calling child_process via dynamic import in the mocked context throws.
    const cp = await import("node:child_process");
    expect(() => (cp as { spawnSync: (...a: unknown[]) => void }).spawnSync("echo", ["hi"])).toThrow(
      /forbidden/,
    );
    expect(calls.length).toBeGreaterThan(0);
    calls.length = 0;
  });
});
