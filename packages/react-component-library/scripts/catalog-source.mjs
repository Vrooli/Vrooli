// One projection of the catalog source tree, shared by the package build and
// the export-map generator. They previously each re-implemented the ledger
// materialization and its bootstrap fallback, and drifted: a version overlaid
// by one was invisible to the other, so the build could compile a component
// the export map then declared unresolvable.
import { cp, mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { homedir } from "node:os";

const scriptsRoot = dirname(fileURLToPath(import.meta.url));
export const packageRoot = scriptsRoot.replace(/[\\/]scripts$/, "");
export const authoredRoot = join(packageRoot, "..", "..", "scenarios", "react-component-library", "library");

const directoriesMatch = async (left, right) => {
  const entries = await readdir(left, { withFileTypes: true });
  for (const entry of entries) {
    const leftPath = join(left, entry.name);
    const rightPath = join(right, entry.name);
    // These files are generated or audit-only artifacts and deliberately do
    // not belong to the release-hash ledger. Excluding them here keeps a
    // derived lock/parity difference from masquerading as authored release
    // drift during package projection.
    if (entry.name === "dependencies.json" || entry.name === "parity.json") continue;
    if (!existsSync(rightPath)) return false;
    if (entry.isDirectory()) {
      if (!await directoriesMatch(leftPath, rightPath)) return false;
      continue;
    }
    if (!(await readFile(leftPath)).equals(await readFile(rightPath))) return false;
  }
  return true;
};

// Lay the authored catalog over the ledger projection.
//
// The authored library is the declared source of truth for source; the ledger
// enumerates and governs releases. Those two disagree today in both
// directions: versions exist on disk with no ledger record, and released rows
// hold content whose bytes differ from the file on disk. Builds used to
// resolve that disagreement by accident — provisioning runs before the API is
// up, so it took the authored tree, while a build against a healthy API took
// ledger content and produced a different package from the same commit.
//
// Authored content wins, deterministically, and both kinds of divergence are
// counted out loud. Silence here is what let an unvalidated ControlBase reach
// every consumer unnoticed.
const overlayAuthoredVersions = async (targetRoot) => {
  const overlaid = [];
  const redivergent = [];
  for (const kind of await readdir(authoredRoot, { withFileTypes: true })) {
    if (!kind.isDirectory()) continue;
    // Preview harnesses are test fixtures consumed by the package's own
    // validation tooling, not published component releases. They have their
    // own manifest/provenance surface and must not be mistaken for ungoverned
    // public library versions during the release projection.
    if (kind.name === "preview-harnesses") continue;
    const kindRoot = join(authoredRoot, kind.name);
    for (const asset of await readdir(kindRoot, { withFileTypes: true })) {
      if (!asset.isDirectory()) continue;
      const versionsRoot = join(kindRoot, asset.name, "versions");
      if (!existsSync(versionsRoot)) continue;
      for (const version of await readdir(versionsRoot, { withFileTypes: true })) {
        if (!version.isDirectory()) continue;
        const relative = join(kind.name, asset.name, "versions", version.name);
        const source = join(versionsRoot, version.name);
        const target = join(targetRoot, relative);
        const present = existsSync(target);
        if (present && await directoriesMatch(source, target)) continue;
        await mkdir(dirname(target), { recursive: true });
        await rm(target, { recursive: true, force: true });
        await cp(source, target, { recursive: true });
        (present ? redivergent : overlaid).push(`${asset.name}@${version.name}`);
      }
    }
  }
  return { overlaid, redivergent };
};

// Scenario setup provisions shared packages before the API process exists.
// The routed host database is still available in that window, so use the
// SQLite file mirror as the bootstrap projection instead of falling back to a
// source tree that is missing every evicted version. This keeps startup
// independent of a recursive `--auto-start` attempt while preserving the API
// as the normal materialization boundary once it is healthy.
const materializeFromLocalLedger = async (targetRoot) => {
  const database = join(homedir(), ".vrooli", "data", "vrooli", "react-component-library", "react-component-library.db");
  if (!existsSync(database)) return false;
  const query = `SELECT v.source_path AS source_path, f.path AS file_path, f.content AS content
FROM component_versions v
JOIN component_version_files f ON f.version_id = v.id
ORDER BY v.source_path, f.path;`;
  const result = spawnSync("sqlite3", ["-json", database, query], { encoding: "utf8", maxBuffer: 64 * 1024 * 1024 });
  if (result.status !== 0) return false;
  let rows;
  try {
    rows = JSON.parse(result.stdout || "[]");
  } catch {
    return false;
  }
  if (!Array.isArray(rows) || rows.length === 0) return false;
  const root = resolve(targetRoot);
  for (const row of rows) {
    if (typeof row.source_path !== "string" || typeof row.file_path !== "string" || typeof row.content !== "string") return false;
    const relativeFile = join(dirname(row.source_path), row.file_path);
    const destination = resolve(targetRoot, relativeFile);
    if (relative(root, destination).startsWith("..")) throw new Error(`ledger file escapes projection root: ${row.file_path}`);
    await mkdir(dirname(destination), { recursive: true });
    await writeFile(destination, row.content);
  }
  return true;
};

/**
 * Project the catalog into targetRoot: ledger first, authored tree during
 * bootstrap, then the unledgered overlay. Returns the overlaid version list.
 * Throws when a healthy API refuses the projection, because that is a real
 * ledger fault rather than a missing endpoint.
 */
export async function projectCatalogSource(targetRoot, { label = "react-component-library" } = {}) {
  await rm(targetRoot, { recursive: true, force: true });
  // The materializer narrates one line per version, which is useful when a
  // projection is being debugged and pure noise in the several hundred lines
  // it adds ahead of every catalog type-check. Capture it and replay it only
  // when the projection actually failed, where it is the diagnostic.
  const materialize = spawnSync("react-component-library", ["versions", "materialize", "--all", "--into", targetRoot], { cwd: packageRoot, encoding: "utf8" });
  if (materialize.status !== 0) {
    if (materialize.stdout) process.stdout.write(materialize.stdout);
    if (materialize.stderr) process.stderr.write(materialize.stderr);
    // Scenario lifecycle provisions shared packages before starting the API,
    // so bootstrap has no ledger endpoint yet; the authored tree covers only
    // that phase. A healthy API that still fails is a projection error.
    const health = spawnSync("curl", ["-fsS", `http://127.0.0.1:${process.env.API_PORT || 17193}/health`], { stdio: "ignore" });
    const boundedMirrorFault = /file mirror is empty|no file mirror rows|evicted version/i.test(`${materialize.stdout || ""}\n${materialize.stderr || ""}`);
    if (boundedMirrorFault && await materializeFromLocalLedger(targetRoot)) {
      console.warn(`${label}: one or more versions could not be materialized; projected readable mirror rows and retained the unreadable versions as named defects`);
    } else if (health.status === 0) {
      throw new Error(`${label}: the version ledger projection failed while the API was healthy`);
    } else if (await materializeFromLocalLedger(targetRoot)) {
      console.warn(`${label}: API unavailable during bootstrap; projecting the local durable ledger mirror`);
    } else if (!boundedMirrorFault) {
      console.warn(`${label}: API and local durable ledger unavailable during bootstrap; projecting the authored tree`);
      await cp(authoredRoot, targetRoot, { recursive: true });
    }
  }
  const { overlaid, redivergent } = await overlayAuthoredVersions(targetRoot);
  if (overlaid.length > 0) {
    console.warn(`${label}: ${overlaid.length} authored version(s) absent from the ledger were overlaid: ${overlaid.sort().join(", ")}`);
    console.warn(`${label}: publish or retire them; a version the ledger does not know has passed no catalog gate.`);
  }
  if (redivergent.length > 0) {
    console.warn(`${label}: ${redivergent.length} version(s) whose ledger content differs from the authored file were rebuilt from the authored file.`);
    console.warn(`${label}: the ledger's copy of those releases is stale; reconcile it before trusting release provenance.`);
  }
  return { overlaid, redivergent };
}
