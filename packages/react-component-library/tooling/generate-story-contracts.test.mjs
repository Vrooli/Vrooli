import test from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, readFile, mkdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { generateStoryContracts } from "./generate-story-contracts.mjs";

test("story contract generation is deterministic and fills only missing contracts", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-story-generator-"));
  const version = join(root, "components", "Button", "versions", "1.0.0");
  await mkdir(version, { recursive: true });
  await writeFile(join(version, "Button.tsx"), "export function Button() { return null; }\n");

  const first = await generateStoryContracts({ root });
  assert.equal(first.written, 1);
  const expected = await readFile(join(version, "story.json"), "utf8");
  const second = await generateStoryContracts({ root });
  assert.equal(second.written, 0);
  assert.equal(await readFile(join(version, "story.json"), "utf8"), expected);
});
