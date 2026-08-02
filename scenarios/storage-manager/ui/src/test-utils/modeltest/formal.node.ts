import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
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
  // generatorSha256 is owned by the flow-verifier scenario, which is
  // external to the consumer scenario. `flow-verifier verify check` is
  // authoritative for generator freshness; this consumer-side helper
  // only verifies that the recorded hash is structurally well-formed.
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
    if (existsSync(path.join(current, ".vrooli", "service.json"))) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) {
      throw new Error("could not locate scenario root containing .vrooli/service.json");
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

const sha256 = (data: string | Buffer): string => createHash("sha256").update(data).digest("hex");

const formatErrors = (errors: readonly string[]): string => errors.map((error) => `  - ${error}`).join("\n");
