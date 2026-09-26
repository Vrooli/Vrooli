import { cp, mkdir, open, readFile, readdir, rm, stat, writeFile, rename } from "node:fs/promises";
import { constants } from "node:fs";
import { createHash, randomUUID } from "node:crypto";
import { join, relative, resolve, sep } from "node:path";
import { homedir } from "node:os";

const manifestName = ".rcl-artifact.json";
const pointerName = "current.json";
const previousName = "previous.json";
const statusName = "runtime-status.json";
const schemaVersion = 1;
const lockWaitMs = 25;
const lockTimeoutMs = 30_000;

function stable(value) {
  if (Array.isArray(value)) return "[" + value.map(stable).join(",") + "]";
  if (value && typeof value === "object") {
    return "{" + Object.keys(value).sort().map((key) => JSON.stringify(key) + ":" + stable(value[key])).join(",") + "}";
  }
  return JSON.stringify(value);
}

function safeId(id) {
  if (!/^sha256:[0-9a-f]{64}$/.test(id)) throw new Error("invalid artifact identity " + JSON.stringify(id));
  return id.slice("sha256:".length);
}

function artifactId(payload) {
  return "sha256:" + createHash("sha256").update(stable(payload)).digest("hex");
}

async function regularFiles(root, base = root) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (entry.name === manifestName) continue;
    const path = join(root, entry.name);
    if (entry.isDirectory()) files.push(...await regularFiles(path, base));
    else if (entry.isFile()) files.push(relative(base, path).split(sep).join("/"));
  }
  return files.sort();
}

async function fileDigest(root, files) {
  const hash = createHash("sha256");
  for (const file of [...files].sort()) {
    const bytes = await readFile(join(root, file));
    hash.update(file.length + ":" + file + ":" + bytes.length + ":");
    hash.update(bytes);
  }
  return "sha256:" + hash.digest("hex");
}

function packageExportTargets(exports) {
  const targets = [];
  const visit = (value, key) => {
    if (typeof value === "string") {
      if (!value.startsWith("./") || value.includes("..")) throw new Error("export " + key + " points outside the package: " + value);
      targets.push({ key, target: value.slice(2) });
      return;
    }
    if (!value || typeof value !== "object") throw new Error("export " + key + " has an invalid target");
    for (const [condition, child] of Object.entries(value)) visit(child, key + " (" + condition + ")");
  };
  for (const [key, value] of Object.entries(exports ?? {})) visit(value, key);
  return targets;
}

async function validateCandidate(root, requiredExports = []) {
  const packagePath = join(root, "package.json");
  const packageInfo = JSON.parse(await readFile(packagePath, "utf8"));
  if (packageInfo.name !== "@vrooli/react-component-library") throw new Error("candidate package name is not @vrooli/react-component-library");
  const files = await regularFiles(root);
  if (!files.includes("package.json")) throw new Error("candidate is incomplete: package.json is missing");
  if (!files.some((file) => file.startsWith("dist/"))) throw new Error("candidate is incomplete: dist output is missing");
  const targets = packageExportTargets(packageInfo.exports);
  const targetSet = new Set(files);
  for (const { key, target } of targets) {
    if (!targetSet.has(target)) throw new Error("candidate export " + key + " points to missing file " + target);
  }
  for (const required of requiredExports) {
    if (!Object.hasOwn(packageInfo.exports ?? {}, required)) throw new Error("candidate exact export " + required + " is missing");
  }
  return { packageInfo, files, outputDigest: await fileDigest(root, files) };
}

function requiredDigest(value, field) {
  if (typeof value !== "string" || !/^sha256:[0-9a-f]{64}$/.test(value)) {
    throw new Error("candidate metadata " + field + " must be a sha256 digest");
  }
  return value;
}

async function exists(path) {
  return stat(path).then(() => true).catch(() => false);
}

async function copyCandidate(candidateRoot, targetRoot) {
  await mkdir(targetRoot, { recursive: true });
  for (const name of ["package.json", "dist"]) {
    const source = join(candidateRoot, name);
    if (await exists(source)) await cp(source, join(targetRoot, name), { recursive: true });
  }
}

export class ArtifactStore {
  constructor(root) {
    this.root = resolve(root);
    this.artifactsRoot = join(this.root, "artifacts");
    this.currentPath = join(this.root, pointerName);
    this.previousPath = join(this.root, previousName);
    this.statusPath = join(this.root, statusName);
    this.lockPath = join(this.root, ".lock");
  }

  async withLock(operation) {
    await mkdir(this.root, { recursive: true });
    const started = Date.now();
    let handle;
    while (!handle) {
      try {
        handle = await open(this.lockPath, constants.O_CREAT | constants.O_EXCL | constants.O_WRONLY, 0o600);
      } catch (error) {
        if (error.code !== "EEXIST" || Date.now() - started >= lockTimeoutMs) throw new Error("artifact store lock timeout: " + error.message);
        await new Promise((resolveDelay) => setTimeout(resolveDelay, lockWaitMs));
      }
    }
    try {
      await handle.writeFile(String(process.pid) + "\n");
      return await operation();
    } finally {
      await handle.close();
      await rm(this.lockPath, { force: true });
    }
  }

  artifactPath(id) {
    return join(this.artifactsRoot, safeId(id));
  }

  async publish(candidateRoot, metadata = {}) {
    return this.withLock(async () => {
      const candidate = resolve(candidateRoot);
      const validated = await validateCandidate(candidate, metadata.requiredExports ?? []);
      const sourceDigest = requiredDigest(metadata.sourceDigest, "sourceDigest");
      const packageLockDigest = requiredDigest(metadata.packageLockDigest, "packageLockDigest");
      const toolchainDigest = requiredDigest(metadata.toolchainDigest, "toolchainDigest");
      const identityPayload = {
        schemaVersion,
        packageName: validated.packageInfo.name,
        packageVersion: validated.packageInfo.version ?? null,
        sourceDigest,
        selectedVersions: metadata.selectedVersions ?? {},
        dependencyClosure: [...(metadata.dependencyClosure ?? [])].sort(),
        packageLockDigest,
        toolchainDigest,
        outputDigest: validated.outputDigest,
      };
      const id = artifactId(identityPayload);
      const destination = this.artifactPath(id);
      if (!(await exists(destination))) {
        const staging = join(this.artifactsRoot, ".staging-" + randomUUID());
        await rm(staging, { recursive: true, force: true });
        await copyCandidate(candidate, staging);
        const manifest = {
          schemaVersion,
          id,
          packageName: validated.packageInfo.name,
          packageVersion: validated.packageInfo.version ?? null,
          sourceDigest: identityPayload.sourceDigest,
          selectedVersions: identityPayload.selectedVersions,
          dependencyClosure: identityPayload.dependencyClosure,
          packageLockDigest: identityPayload.packageLockDigest,
          toolchainDigest: identityPayload.toolchainDigest,
          outputDigest: validated.outputDigest,
          files: validated.files,
          publishedAt: new Date().toISOString(),
        };
        await writeFile(join(staging, manifestName), JSON.stringify(manifest, null, 2) + "\n", { mode: 0o644 });
        await mkdir(this.artifactsRoot, { recursive: true });
        await rename(staging, destination);
      }
      if (process.env.RCL_ARTIFACT_FAIL_STAGE === "before-select") {
        throw new Error("injected publication interruption before current selection");
      }
      await this.selectUnlocked(id);
      return { id, manifest: await this.readManifest(id) };
    });
  }

  async readManifest(id) {
    return JSON.parse(await readFile(join(this.artifactPath(id), manifestName), "utf8"));
  }

  async verify(id) {
    try {
      const safe = safeId(id);
      const root = this.artifactPath(id);
      const manifest = await this.readManifest(id);
      if (manifest.id !== id || manifest.schemaVersion !== schemaVersion) return { valid: false, id, reason: "manifest identity/schema mismatch" };
      const packageInfo = JSON.parse(await readFile(join(root, "package.json"), "utf8"));
      const validated = await validateCandidate(root, Object.keys(packageInfo.exports ?? {}));
      if (stable(manifest.files ?? []) !== stable(validated.files)) return { valid: false, id, reason: "artifact file manifest mismatch" };
      if (validated.outputDigest !== manifest.outputDigest) return { valid: false, id, reason: "artifact output digest mismatch" };
      const expectedId = artifactId({
        schemaVersion,
        packageName: manifest.packageName,
        packageVersion: manifest.packageVersion,
        sourceDigest: manifest.sourceDigest,
        selectedVersions: manifest.selectedVersions ?? {},
        dependencyClosure: [...(manifest.dependencyClosure ?? [])].sort(),
        packageLockDigest: manifest.packageLockDigest,
        toolchainDigest: manifest.toolchainDigest,
        outputDigest: manifest.outputDigest,
      });
      if (expectedId !== id) return { valid: false, id, reason: "artifact identity digest mismatch" };
      return { valid: true, id: "sha256:" + safe, manifest };
    } catch (error) {
      return { valid: false, id, reason: error.message };
    }
  }

  async select(id) {
    return this.withLock(() => this.selectUnlocked(id));
  }

  async selectUnlocked(id) {
    const verification = await this.verify(id);
    if (!verification.valid) throw new Error("cannot select artifact " + id + ": " + verification.reason);
    const current = await this.readCurrent();
    if (current?.id === id) return current;
    if (current) await writeFile(this.previousPath, JSON.stringify(current, null, 2) + "\n");
    const temporary = this.currentPath + "." + randomUUID() + ".tmp";
    await writeFile(temporary, JSON.stringify({ schemaVersion, id, selectedAt: new Date().toISOString() }, null, 2) + "\n", { mode: 0o644 });
    await rename(temporary, this.currentPath);
    return this.readCurrent();
  }

  async readCurrent() {
    try {
      return JSON.parse(await readFile(this.currentPath, "utf8"));
    } catch (error) {
      if (error.code === "ENOENT") return null;
      throw error;
    }
  }

  async readCurrentVerified() {
    const selection = await this.readCurrent();
    if (!selection?.id) throw new Error("no selected react-component-library artifact");
    const verification = await this.verify(selection.id);
    if (!verification.valid) throw new Error("selected artifact is invalid: " + verification.reason);
    return { selection, verification };
  }

  async requireCompatible(id, requiredExports = []) {
    const verification = await this.verify(id);
    if (!verification.valid) throw new Error("artifact is invalid: " + verification.reason);
    const packageInfo = JSON.parse(await readFile(join(this.artifactPath(id), "package.json"), "utf8"));
    const missing = requiredExports.filter((key) => !Object.hasOwn(packageInfo.exports ?? {}, key));
    if (missing.length > 0) throw new Error("artifact is missing exact exports: " + missing.join(", "));
    return verification;
  }

  async writeStatus(status) {
    await mkdir(this.root, { recursive: true });
    const temporary = this.statusPath + "." + randomUUID() + ".tmp";
    await writeFile(temporary, JSON.stringify(status, null, 2) + "\n", { mode: 0o644 });
    await rename(temporary, this.statusPath);
    return status;
  }

  async readStatus() {
    try {
      return JSON.parse(await readFile(this.statusPath, "utf8"));
    } catch (error) {
      if (error.code === "ENOENT") return null;
      throw error;
    }
  }

  async prune(maxArtifacts = 3) {
    if (!Number.isInteger(maxArtifacts) || maxArtifacts < 1) throw new Error("maxArtifacts must be a positive integer");
    return this.withLock(async () => {
      const current = await this.readCurrent();
      let previous = null;
      try { previous = JSON.parse(await readFile(this.previousPath, "utf8")); } catch (error) { if (error.code !== "ENOENT") throw error; }
      const protectedIds = new Set([current?.id, previous?.id].filter(Boolean));
      const entries = (await readdir(this.artifactsRoot, { withFileTypes: true }).catch((error) => {
        if (error.code === "ENOENT") return [];
        throw error;
      })).filter((entry) => entry.isDirectory() && /^[0-9a-f]{64}$/.test(entry.name));
      const records = [];
      for (const entry of entries) {
        const id = "sha256:" + entry.name;
        const manifest = await this.readManifest(id).catch(() => null);
        records.push({ id, publishedAt: manifest?.publishedAt ?? "", path: join(this.artifactsRoot, entry.name) });
      }
      records.sort((left, right) => left.publishedAt.localeCompare(right.publishedAt));
      const keep = new Set([...protectedIds, ...records.slice(-maxArtifacts).map((record) => record.id)]);
      const removed = [];
      for (const record of records) {
        if (keep.has(record.id)) continue;
        await rm(record.path, { recursive: true, force: true });
        removed.push(record.id);
      }
      return { kept: records.filter((record) => keep.has(record.id)).map((record) => record.id), removed };
    });
  }

  async materialize(id, targetRoot) {
    return this.withLock(async () => {
      const verification = await this.verify(id);
      if (!verification.valid) throw new Error("cannot materialize artifact " + id + ": " + verification.reason);
      const destination = resolve(targetRoot);
      const source = this.artifactPath(id);
      const temporary = destination + ".rcl-materialize-" + randomUUID();
      await rm(temporary, { recursive: true, force: true });
      await copyCandidate(source, temporary);
      await mkdir(destination, { recursive: true });
      const oldDist = join(destination, "dist");
      const retired = join(destination, "dist.previous");
      await rm(retired, { recursive: true, force: true });
      if (await exists(oldDist)) await rename(oldDist, retired);
      await rename(join(temporary, "dist"), oldDist);
      await rename(join(temporary, "package.json"), join(destination, "package.json"));
      await rm(retired, { recursive: true, force: true });
      await rm(temporary, { recursive: true, force: true });
      return verification;
    });
  }
}

export function defaultArtifactStoreRoot() {
  return process.env.VROOLI_RCL_ARTIFACT_ROOT
    || join(process.env.VROOLI_RUNTIME_HOME || join(homedir(), ".vrooli"), "artifacts", "react-component-library");
}

export async function directoryDigest(root, ignored = new Set()) {
  const files = (await regularFiles(resolve(root))).filter((file) => !ignored.has(file));
  return fileDigest(root, files);
}

export { artifactId, fileDigest, validateCandidate };
