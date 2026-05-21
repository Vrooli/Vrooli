import { createHash } from "node:crypto";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import {
  validateFormalArtifactFreshFromFiles,
} from "./formal.node";
import type { FormalArtifact } from "./formal";

const contractPath = "ui/src/features/example/model.flow.json";
const modelPath = "ui/src/features/example/model.qnt";
const generatorPath = "flow-verifier";

describe("Node formal modeltest helpers", () => {
  it("recomputes contract, model, and generator hashes from files", () => {
    const root = makeScenarioTree();
    try {
      const artifact = validArtifact({
        contractSha256: fileSHA256(path.join(root, contractPath)),
        modelSha256: fileSHA256(path.join(root, modelPath)),
        generatorSha256: treeSHA256(path.join(root, generatorPath)),
      });

      expect(validateFormalArtifactFreshFromFiles(artifact, {
        contractPath,
        modelPath,
        generatorPath,
      }, { scenarioRoot: root })).toEqual([]);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });

  it("reports recomputed hash mismatches", () => {
    const root = makeScenarioTree();
    try {
      const artifact = validArtifact({
        contractSha256: "a".repeat(64),
        modelSha256: "b".repeat(64),
        generatorSha256: "c".repeat(64),
      });

      const errors = validateFormalArtifactFreshFromFiles(artifact, {
        contractPath,
        modelPath,
        generatorPath,
      }, { scenarioRoot: root });

      expect(errors.join("\n")).toContain("contractSha256=");
      expect(errors.join("\n")).toContain("modelSha256=");
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
});

const makeScenarioTree = (): string => {
  const root = mkdtempSync(path.join(os.tmpdir(), "formal-node-"));
  write(root, contractPath, "{\"schemaVersion\":3}\n");
  write(root, modelPath, "module Example\n");
  write(root, path.join(generatorPath, "go.mod"), "module example\n");
  write(root, path.join(generatorPath, "main.go"), "package main\n");
  write(root, path.join(generatorPath, "flow.schema.json"), "{}\n");
  write(root, path.join(generatorPath, "internal", "tool_test.go"), "package internal\n");
  write(root, path.join(generatorPath, "internal", "model.qnt"), "ignored\n");
  return root;
};

const write = (root: string, relative: string, body: string): void => {
  const file = path.join(root, relative);
  mkdirSync(path.dirname(file), { recursive: true });
  writeFileSync(file, body);
};

const validArtifact = (sourceHashes: {
  readonly contractSha256: string;
  readonly modelSha256: string;
  readonly generatorSha256: string;
}): FormalArtifact => ({
  schemaVersion: 6,
  flowId: "example.flow",
  source: {
    contractPath,
    contractSha256: sourceHashes.contractSha256,
    generatorPath,
    generatorSha256: sourceHashes.generatorSha256,
    generatorVersion: 2,
    modelPath,
    modelSha256: sourceHashes.modelSha256,
    quintVersion: "0.32.0",
    verificationBackend: "apalache",
  },
  commands: {
    typecheck: ["quint", "typecheck", modelPath],
    test: ["quint", "test", modelPath],
    verify: ["quint", "verify", modelPath],
    run: ["quint", "run", modelPath],
  },
  states: ["idle"],
  events: ["tick"],
  transitions: [{ from: "idle", event: "tick", to: "idle", wantError: true }],
  namedTraces: [{ name: "idle", initial: "idle", steps: [] }],
  generatedTraces: [{ name: "generated", initial: "idle", steps: [] }],
  invariants: ["TypeOK"],
  generatedChecks: ["transitionTable"],
  coverage: {
    transitionMatrixComplete: true,
    terminalTransitionsChecked: true,
    namedTraces: {
      allStatesCovered: true,
      allEventsCovered: true,
      coveredStates: ["idle"],
      coveredEvents: ["tick"],
    },
    generatedTraces: {
      allStatesCovered: true,
      allEventsCovered: true,
      coveredStates: ["idle"],
      coveredEvents: ["tick"],
      coveredPairs: ["idle/tick"],
      allPairsCovered: true,
    },
  },
  checks: {
    typechecked: true,
    tested: true,
    verified: true,
    generatedFromContract: true,
    generatedFromModel: true,
  },
});

const fileSHA256 = (file: string): string => sha256(readFileSync(file));

const treeSHA256 = (root: string): string => {
  const parts = [
    `flow.schema.json\0${fileSHA256(path.join(root, "flow.schema.json"))}`,
    `go.mod\0${fileSHA256(path.join(root, "go.mod"))}`,
    `main.go\0${fileSHA256(path.join(root, "main.go"))}`,
  ].sort();
  return sha256(parts.join("\n"));
};

const sha256 = (data: string | Buffer): string => createHash("sha256").update(data).digest("hex");
