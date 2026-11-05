import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { applyRewrite } from "../src/rewrite.js";

let tmpDir: string;

function write(rel: string, body: string): string {
  const abs = path.join(tmpDir, rel);
  fs.mkdirSync(path.dirname(abs), { recursive: true });
  fs.writeFileSync(abs, body);
  return abs;
}

beforeEach(() => {
  tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "tcg-sidecar-rw-"));
  write(
    "tsconfig.json",
    JSON.stringify({
      compilerOptions: {
        target: "ES2022",
        module: "NodeNext",
        moduleResolution: "NodeNext",
        strict: true,
        jsx: "react-jsx",
      },
      include: ["src/**/*"],
    }),
  );
});

afterEach(() => {
  fs.rmSync(tmpDir, { recursive: true, force: true });
});

describe("applyRewrite", () => {
  it("moves a file and rewrites imports together", async () => {
    write("src/util.ts", `export const U = 1;\n`);
    write("src/main.ts", `import { U } from "./util";\nexport const M = U;\n`);

    const results = await applyRewrite({
      projectPath: tmpDir,
      operations: [
        {
          file_move: {
            from_path: path.join(tmpDir, "src/util.ts"),
            to_path: path.join(tmpDir, "src/lib/util.ts"),
          },
        },
      ],
    });

    expect(results).toHaveLength(1);
    expect(results[0]!.status).toBe("OPERATION_STATUS_OK");

    // File should be at new path
    expect(fs.existsSync(path.join(tmpDir, "src/lib/util.ts"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, "src/util.ts"))).toBe(false);
    // ts-morph rewrites the importing file's specifier on move
    const main = fs.readFileSync(path.join(tmpDir, "src/main.ts"), "utf8");
    expect(main).toMatch(/from\s+["']\.\/lib\/util["']/);
  });

  it("import_rewrite touches all files referencing the old specifier", async () => {
    write("src/util.ts", `export const U = 1;\n`);
    write("src/a.ts", `import { U } from "./util";\nexport const A = U;\n`);
    write("src/b.ts", `import { U } from "./util";\nexport const B = U;\n`);

    const results = await applyRewrite({
      projectPath: tmpDir,
      operations: [
        { import_rewrite: { old_path: "./util", new_path: "./util-renamed" } },
      ],
    });

    expect(results[0]!.status).toBe("OPERATION_STATUS_OK");
    expect(results[0]!.message).toBe("rewrote imports in 2 file(s)");
    expect(fs.readFileSync(path.join(tmpDir, "src/a.ts"), "utf8")).toMatch(
      /from\s+["']\.\/util-renamed["']/,
    );
    expect(fs.readFileSync(path.join(tmpDir, "src/b.ts"), "utf8")).toMatch(
      /from\s+["']\.\/util-renamed["']/,
    );
  });

  it("returns FAILED for a move whose source file is not in the project", async () => {
    write("src/main.ts", `export const M = 1;\n`);
    const results = await applyRewrite({
      projectPath: tmpDir,
      operations: [
        {
          file_move: {
            from_path: path.join(tmpDir, "src/missing.ts"),
            to_path: path.join(tmpDir, "src/elsewhere.ts"),
          },
        },
      ],
    });
    expect(results[0]!.status).toBe("OPERATION_STATUS_FAILED");
    expect(results[0]!.message).toMatch(/not found/);
  });

  it("returns FAILED when neither file_move nor import_rewrite is set", async () => {
    write("src/main.ts", `export const M = 1;\n`);
    const results = await applyRewrite({
      projectPath: tmpDir,
      operations: [{}],
    });
    expect(results[0]!.status).toBe("OPERATION_STATUS_FAILED");
  });
});
