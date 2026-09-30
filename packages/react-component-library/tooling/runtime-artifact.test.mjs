import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import assert from "node:assert/strict";
import { ArtifactStore } from "./runtime-artifact.mjs";

const digest = (letter) => `sha256:${letter.repeat(64)}`;

async function candidateRoot() {
  const root = await mkdtemp(join(tmpdir(), "rcl-candidate-"));
  await mkdir(join(root, "dist", "exports", "Button"), { recursive: true });
  await writeFile(join(root, "package.json"), JSON.stringify({
    name: "@vrooli/react-component-library",
    exports: {
      ".": { types: "./dist/exports/index.d.ts", import: "./dist/exports/index.js" },
      "./Button/1.0.0": { types: "./dist/exports/Button/1.0.0.d.ts", import: "./dist/exports/Button/1.0.0.js" },
    },
  }, null, 2));
  await writeFile(join(root, "dist", "exports", "index.js"), "export * from './Button/1.0.0.js';\n");
  await writeFile(join(root, "dist", "exports", "index.d.ts"), "export * from './Button/1.0.0';\n");
  await writeFile(join(root, "dist", "exports", "Button", "1.0.0.js"), "export const Button = () => null;\n");
  await writeFile(join(root, "dist", "exports", "Button", "1.0.0.d.ts"), "export declare const Button: () => null;\n");
  return root;
}

test("publishes a complete immutable artifact and selects it atomically", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const candidate = await candidateRoot();
  const store = new ArtifactStore(root);

  const published = await store.publish(candidate, {
    sourceDigest: digest("a"),
    selectedVersions: { Button: "1.0.0" },
    dependencyClosure: ["Button@1.0.0"],
    packageLockDigest: digest("b"),
    toolchainDigest: digest("c"),
  });

  assert.match(published.id, /^sha256:[0-9a-f]{64}$/);
  assert.equal((await store.readCurrent()).id, published.id);
  const verification = await store.verify(published.id);
  assert.equal(verification.valid, true);
  assert.equal(verification.id, published.id);
  assert.equal(await readFile(join(root, "artifacts", published.id.slice("sha256:".length), "package.json"), "utf8").then(JSON.parse).then((pkg) => pkg.name), "@vrooli/react-component-library");
});

test("rejects incomplete candidates without changing the selected artifact", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const good = await candidateRoot();
  const incomplete = await candidateRoot();
  const store = new ArtifactStore(root);
  const first = await store.publish(good, { sourceDigest: digest("d"), packageLockDigest: digest("e"), toolchainDigest: digest("f") });
  const selectedBefore = await store.readCurrent();
  await rm(join(incomplete, "dist", "exports", "Button", "1.0.0.js"), { force: true });

  await assert.rejects(
    store.publish(incomplete, { sourceDigest: digest("1"), packageLockDigest: digest("2"), toolchainDigest: digest("3"), requiredExports: ["./Button/1.0.0"] }),
    /candidate.*(integrity|export|complete)/i,
  );
  assert.equal((await store.readCurrent()).id, selectedBefore.id);
  const verification = await store.verify(first.id);
  assert.equal(verification.valid, true);
  assert.equal(verification.id, first.id);
});

test("detects corruption and refuses to select a corrupt artifact", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const candidate = await candidateRoot();
  const store = new ArtifactStore(root);
  const first = await store.publish(candidate, { sourceDigest: digest("4"), packageLockDigest: digest("5"), toolchainDigest: digest("6") });
  const artifactRoot = join(root, "artifacts", first.id.slice("sha256:".length));
  await writeFile(join(artifactRoot, "dist", "exports", "Button", "1.0.0.js"), "tampered");

  const verification = await store.verify(first.id);
  assert.equal(verification.valid, false);
  await assert.rejects(store.select(first.id), /artifact.*(digest|integrity|corrupt|invalid)/i);
});

test("retention never removes the current or rollback artifact", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const store = new ArtifactStore(root);
  const ids = [];
  for (const letter of ["7", "8", "9"]) {
    ids.push((await store.publish(await candidateRoot(), {
      sourceDigest: digest(letter),
      packageLockDigest: digest(letter),
      toolchainDigest: digest(letter),
    })).id);
  }
  const result = await store.prune(1);
  assert.ok(result.kept.includes(ids[1]), "previous selection remains protected");
  assert.ok(result.kept.includes(ids[2]), "current selection remains protected");
  assert.ok(result.removed.includes(ids[0]), "old unprotected artifact is pruned");
  assert.equal((await store.verify(ids[0])).valid, false);
});

test("publication interruption leaves the previous current pointer intact", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const store = new ArtifactStore(root);
  const first = await store.publish(await candidateRoot(), {
    sourceDigest: digest("a"), packageLockDigest: digest("b"), toolchainDigest: digest("c"),
  });
  process.env.RCL_ARTIFACT_FAIL_STAGE = "before-select";
  try {
    await assert.rejects(store.publish(await candidateRoot(), {
      sourceDigest: digest("d"), packageLockDigest: digest("e"), toolchainDigest: digest("f"),
    }), /publication interruption/);
  } finally {
    delete process.env.RCL_ARTIFACT_FAIL_STAGE;
  }
  assert.equal((await store.readCurrent()).id, first.id);
  assert.equal((await store.verify(first.id)).valid, true);
});

test("compatibility preflight fails closed for a missing exact export", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const store = new ArtifactStore(root);
  const published = await store.publish(await candidateRoot(), {
    sourceDigest: digest("a"), packageLockDigest: digest("b"), toolchainDigest: digest("c"),
  });

  await assert.rejects(
    store.requireCompatible(published.id, ["./Button/9.9.9"]),
    /missing exact exports.*\.\/Button\/9\.9\.9/,
  );
  assert.equal((await store.readCurrent()).id, published.id);
});

test("runtime status is replaced atomically and records the selected artifact", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const store = new ArtifactStore(root);
  const published = await store.publish(await candidateRoot(), {
    sourceDigest: digest("a"), packageLockDigest: digest("b"), toolchainDigest: digest("c"),
  });
  const status = {
    status: "degraded",
    packageName: "@vrooli/react-component-library",
    candidateIdentity: "candidate:test",
    candidateArtifact: null,
    selectedArtifact: published.id,
    reason: "candidate compilation failed",
    compatibility: "verified",
    updatedAt: new Date().toISOString(),
  };
  await store.writeStatus(status);
  assert.deepEqual(await store.readStatus(), status);
});

test("materialization leaves the file dependency facade coherent", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-artifacts-"));
  const target = await mkdtemp(join(tmpdir(), "rcl-facade-"));
  const store = new ArtifactStore(root);
  const published = await store.publish(await candidateRoot(), {
    sourceDigest: digest("a"), packageLockDigest: digest("b"), toolchainDigest: digest("c"),
  });
  await mkdir(join(target, "dist"), { recursive: true });
  await writeFile(join(target, "package.json"), JSON.stringify({ name: "stale" }));
  await writeFile(join(target, "dist", "old.js"), "old\n");

  await store.materialize(published.id, target);
  const packageInfo = JSON.parse(await readFile(join(target, "package.json"), "utf8"));
  assert.equal(packageInfo.name, "@vrooli/react-component-library");
  assert.ok(Object.hasOwn(packageInfo.exports, "./Button/1.0.0"));
  assert.equal(await readFile(join(target, "dist", "exports", "Button", "1.0.0.js"), "utf8"), "export const Button = () => null;\n");
});
