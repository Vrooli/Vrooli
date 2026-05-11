import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  validateFormalArtifactFresh,
  type FormalArtifact,
  type FormalArtifactFreshExpectation,
} from "./formal";

export interface FormalArtifactFreshFileOptions {
  readonly scenarioRoot?: string | URL;
}

export const validateFormalArtifactFreshFromFiles = (
  artifact: FormalArtifact,
  expected: FormalArtifactFreshExpectation,
  options: FormalArtifactFreshFileOptions = {},
): string[] => {
  const errors = validateFormalArtifactFresh(artifact, expected);
  const root = resolveScenarioRoot(options.scenarioRoot);
  compareHash(errors, "contractSha256", artifact.source.contractSha256, () => fileSHA256(path.join(root, artifact.source.contractPath)));
  compareHash(errors, "modelSha256", artifact.source.modelSha256, () => fileSHA256(path.join(root, artifact.source.modelPath)));
  compareHash(errors, "generatorSha256", artifact.source.generatorSha256, () => treeSHA256(path.join(root, artifact.source.generatorPath)));
  return errors;
};

export const assertFormalArtifactFreshFromFiles = (
  artifact: FormalArtifact,
  expected: FormalArtifactFreshExpectation,
  options: FormalArtifactFreshFileOptions = {},
): void => {
  const errors = validateFormalArtifactFreshFromFiles(artifact, expected, options);
  if (errors.length > 0) {
    throw new Error(`formal artifact is stale or incomplete:\n${formatErrors(errors)}`);
  }
};

const resolveScenarioRoot = (root?: string | URL): string => {
  if (root !== undefined) {
    return path.resolve(root instanceof URL ? fileURLToPath(root) : root);
  }
  let current = process.cwd();
  for (;;) {
    if (existsSync(path.join(current, "tools", "temporal-model", "go.mod"))) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) {
      throw new Error("could not locate scenario root containing tools/temporal-model/go.mod");
    }
    current = parent;
  }
};

const fileSHA256 = (filePath: string): string => sha256(readFileSync(filePath));

const compareHash = (errors: string[], label: string, artifactHash: string, recompute: () => string): void => {
  let freshHash: string;
  try {
    freshHash = recompute();
  } catch (error) {
    errors.push(`formal artifact ${label} could not be recomputed: ${String(error)}`);
    return;
  }
  if (artifactHash !== freshHash) {
    errors.push(`formal artifact ${label}=${artifactHash}, recomputed ${freshHash}`);
  }
};

const treeSHA256 = (root: string): string => {
  const parts: string[] = [];
  walk(root, root, parts);
  parts.sort();
  return sha256(parts.join("\n"));
};

const walk = (root: string, current: string, parts: string[]): void => {
  for (const entry of readdirSync(current, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const absolute = path.join(current, entry.name);
    const relative = path.relative(root, absolute).split(path.sep).join("/");
    if (entry.isDirectory()) {
      if (ignoredDirs.has(entry.name)) {
        continue;
      }
      walk(root, absolute, parts);
      continue;
    }
    if (!includedGeneratorFile(relative)) {
      continue;
    }
    parts.push(`${relative}\0${fileSHA256(absolute)}`);
  }
};

const includedGeneratorFile = (relative: string): boolean => {
  if (
    relative.startsWith("testdata/")
    || relative.endsWith("_test.go")
    || relative.endsWith(".formal.generated.json")
    || relative.endsWith(".qnt")
  ) {
    return false;
  }
  return relative.endsWith(".go") || relative === "go.mod" || relative.endsWith(".schema.json");
};

const ignoredDirs = new Set([".git", "node_modules", "dist", "build", "coverage", "_apalache-out"]);

const sha256 = (data: string | Buffer): string => createHash("sha256").update(data).digest("hex");

const formatErrors = (errors: readonly string[]): string => errors.map((error) => `  - ${error}`).join("\n");
