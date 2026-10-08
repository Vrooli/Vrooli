import test from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile, readFile, rm, access } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { generateLocks } from "./generate-locks.mjs";

test("active drafts receive locks and resolve children exclusively to releases", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-draft-locks-"));
  try {
    for (const name of ["Child", "Parent"]) {
      const base = join(root, "components", name);
      await mkdir(base, { recursive: true });
      await writeFile(join(base, "component.json"), JSON.stringify({ libraryId: `react-component-library:${name}`, latest: "1.0.0", draft: "2.0.0-draft.1", entry: `${name}.tsx` }));
      for (const version of ["1.0.0", "2.0.0-draft.1", "3.0.0-abandoned.1"]) {
        const dir = join(base, "versions", version);
        await mkdir(dir, { recursive: true });
        await writeFile(join(dir, `${name}.tsx`), name === "Parent" ? 'import { Child } from "@vrooli/react-component-library/Child/1"; export const Parent = Child;' : 'export const Child = () => null;');
      }
    }
    const before = await generateLocks({ libraryRoot: root, check: true });
    assert.equal(before.stale.length, 4);
    const result = await generateLocks({ libraryRoot: root });
    assert.equal(result.written, 4);
    const path = join(root, "components/Parent/versions/2.0.0-draft.1/dependencies.json");
    const lock = JSON.parse(await readFile(path, "utf8"));
    assert.equal(lock.version, "2.0.0-draft.1");
    assert.equal(lock.dependencies[0].observed, "1.0.0");
    await assert.rejects(access(join(root, "components/Parent/versions/3.0.0-abandoned.1/dependencies.json")));
    assert.deepEqual((await generateLocks({ libraryRoot: root, check: true })).stale, []);
    const releasedPath = join(root, "components/Parent/versions/1.0.0/dependencies.json");
    const releasedBytes = await readFile(releasedPath, "utf8");
    const newerChild = join(root, "components/Child/versions/1.1.0");
    await mkdir(newerChild);
    await writeFile(join(newerChild, "Child.tsx"), "export const Child = () => null;");
    await generateLocks({ libraryRoot: root });
    assert.equal(await readFile(releasedPath, "utf8"), releasedBytes, "a new child release must not rewrite a released parent");
    assert.equal(JSON.parse(await readFile(path, "utf8")).dependencies[0].observed, "1.1.0", "the active draft can adopt the new child");
  } finally { await rm(root, { recursive: true, force: true }); }
});
