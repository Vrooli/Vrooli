#!/usr/bin/env node
import { createHash } from "node:crypto";
import { mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const scriptPath = fileURLToPath(import.meta.url);
const templateRoot = path.resolve(path.dirname(scriptPath), "../..");

const flows = [
  {
    flowId: "notes.attachment-upload.api",
    modelPath: "api/internal/notes/attachment_upload_workflow.qnt",
    artifactPath: "api/internal/notes/attachment_upload_workflow.formal.generated.json",
    invariant: "TypeOK",
    seed: "2026051001",
    maxSteps: 6,
    traceCount: 8,
    states: ["received", "bytes_stored", "metadata_recorded", "failed"],
    events: ["store_bytes", "record_metadata", "fail"],
    tagToState: {
      Received: "received",
      BytesStored: "bytes_stored",
      MetadataRecorded: "metadata_recorded",
      Failed: "failed",
    },
    tagToEvent: {
      StoreBytes: "store_bytes",
      RecordMetadata: "record_metadata",
      Fail: "fail",
    },
    transitions: [
      ["received", "store_bytes", "bytes_stored", false],
      ["received", "record_metadata", "received", true],
      ["received", "fail", "failed", false],
      ["bytes_stored", "store_bytes", "bytes_stored", true],
      ["bytes_stored", "record_metadata", "metadata_recorded", false],
      ["bytes_stored", "fail", "failed", false],
      ["metadata_recorded", "store_bytes", "metadata_recorded", true],
      ["metadata_recorded", "record_metadata", "metadata_recorded", true],
      ["metadata_recorded", "fail", "metadata_recorded", true],
      ["failed", "store_bytes", "failed", true],
      ["failed", "record_metadata", "failed", true],
      ["failed", "fail", "failed", true],
    ],
  },
  {
    flowId: "notes.attachment-upload.ui",
    modelPath: "ui/src/features/notes/AttachmentUploadWorkflow.qnt",
    artifactPath: "ui/src/features/notes/AttachmentUploadWorkflow.formal.generated.json",
    invariant: "TypeOK",
    seed: "2026051002",
    maxSteps: 7,
    traceCount: 10,
    states: ["idle", "selected", "uploading", "succeeded", "failed"],
    events: ["select", "start", "succeed", "fail", "reset"],
    tagToState: {
      Idle: "idle",
      Selected: "selected",
      Uploading: "uploading",
      Succeeded: "succeeded",
      Failed: "failed",
    },
    tagToEvent: {
      Select: "select",
      Start: "start",
      Succeed: "succeed",
      Fail: "fail",
      Reset: "reset",
    },
    transitions: [
      ["idle", "select", "selected", false],
      ["idle", "start", "idle", true],
      ["idle", "succeed", "idle", true],
      ["idle", "fail", "idle", true],
      ["idle", "reset", "idle", false],
      ["selected", "select", "selected", false],
      ["selected", "start", "uploading", false],
      ["selected", "succeed", "selected", true],
      ["selected", "fail", "selected", true],
      ["selected", "reset", "idle", false],
      ["uploading", "select", "selected", false],
      ["uploading", "start", "uploading", true],
      ["uploading", "succeed", "succeeded", false],
      ["uploading", "fail", "failed", false],
      ["uploading", "reset", "idle", false],
      ["succeeded", "select", "selected", false],
      ["succeeded", "start", "succeeded", true],
      ["succeeded", "succeed", "succeeded", true],
      ["succeeded", "fail", "succeeded", true],
      ["succeeded", "reset", "idle", false],
      ["failed", "select", "selected", false],
      ["failed", "start", "uploading", false],
      ["failed", "succeed", "failed", true],
      ["failed", "fail", "failed", true],
      ["failed", "reset", "idle", false],
    ],
  },
];

const args = new Set(process.argv.slice(2));
const check = args.has("--check");
const flowArgIndex = process.argv.indexOf("--flow");
const selectedFlowId = flowArgIndex >= 0 ? process.argv[flowArgIndex + 1] : "";

if (args.has("--help")) {
  console.log("Usage: node tools/temporal-model/generate.mjs [--check] [--flow <flow-id>]");
  process.exit(0);
}

if (flowArgIndex >= 0 && !selectedFlowId) {
  fail("--flow requires a flow id");
}

const selectedFlows = selectedFlowId ? flows.filter((flow) => flow.flowId === selectedFlowId) : flows;
if (selectedFlows.length === 0) {
  fail(`unknown flow id ${selectedFlowId}`);
}

const quintVersion = run(["quint", "--version"], { quiet: true }).stdout.trim();
if (!quintVersion) {
  fail("quint --version returned an empty version");
}

let wrote = 0;
for (const flow of selectedFlows) {
  const artifact = generateArtifact(flow, quintVersion);
  const artifactPath = path.join(templateRoot, flow.artifactPath);
  const next = canonicalJson(artifact);

  if (check) {
    const current = readFileSync(artifactPath, "utf8");
    if (current !== next) {
      fail(`${flow.artifactPath} is stale. Run node tools/temporal-model/generate.mjs --flow ${flow.flowId}`);
    }
    console.log(`fresh ${flow.flowId}`);
    continue;
  }

  writeFileSync(artifactPath, next);
  wrote += 1;
  console.log(`wrote ${flow.artifactPath}`);
}

if (!check) {
  console.log(`generated ${wrote} formal temporal artifact(s)`);
}

function generateArtifact(flow, quintVersion) {
  const modelAbs = path.join(templateRoot, flow.modelPath);
  const tempDir = mkdtempSync(path.join(tmpdir(), "react-vite-temporal-model-"));
  const itfPattern = path.join(tempDir, `${flow.flowId.replaceAll(".", "-")}_{seq}.itf.json`);

  const commands = {
    typecheck: ["quint", "typecheck", flow.modelPath],
    test: ["quint", "test", flow.modelPath, "--seed", flow.seed],
    verify: ["quint", "verify", flow.modelPath, "--invariant", flow.invariant, "--max-steps", String(flow.maxSteps)],
    run: [
      "quint",
      "run",
      flow.modelPath,
      "--mbt",
      "--seed",
      flow.seed,
      "--max-samples",
      String(flow.traceCount),
      "--n-traces",
      String(flow.traceCount),
      "--max-steps",
      String(flow.maxSteps),
      "--out-itf",
      "<temp-itf-pattern>",
    ],
  };

  try {
    run(commands.typecheck);
    run(commands.test);
    run(commands.verify);
    run(commands.run.map((part) => (part === "<temp-itf-pattern>" ? itfPattern : part)));

    return stable({
      schemaVersion: 1,
      flowId: flow.flowId,
      source: {
        modelPath: flow.modelPath,
        modelSha256: sha256(readFileSync(modelAbs)),
        quintVersion,
      },
      commands,
      states: flow.states,
      events: flow.events,
      transitions: flow.transitions.map(([from, event, to, wantError]) => ({ from, event, to, wantError })),
      traces: normalizeTraces(flow, tempDir),
      checks: {
        typechecked: true,
        tested: true,
        verified: true,
        generatedFromModel: true,
      },
    });
  } finally {
    rmSync(tempDir, { force: true, recursive: true });
  }
}

function run(command, options = {}) {
  const [bin, ...cmdArgs] = command;
  const result = spawnSync(bin, cmdArgs, {
    cwd: templateRoot,
    encoding: "utf8",
    stdio: options.quiet ? "pipe" : ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) {
    const rendered = command.join(" ");
    const stderr = result.stderr?.trim();
    const stdout = result.stdout?.trim();
    fail([`command failed: ${rendered}`, stdout, stderr].filter(Boolean).join("\n"));
  }
  return result;
}

function normalizeTraces(flow, tempDir) {
  const files = readdirSync(tempDir).sort();

  return files.map((file, index) => {
    const raw = JSON.parse(readFileSync(path.join(tempDir, file), "utf8"));
    const states = raw.states ?? [];
    if (states.length === 0) {
      fail(`${file} contains no ITF states`);
    }

    const initial = decodeTag(states[0].status, flow.tagToState, "status", file);
    const steps = states.slice(1).map((state) => ({
      event: decodeTag(state.event, flow.tagToEvent, "event", file),
      want: decodeTag(state.status, flow.tagToState, "status", file),
      wantError: Boolean(state.rejected),
    }));

    return {
      name: `generated_model_${String(index + 1).padStart(3, "0")}`,
      initial,
      steps,
    };
  });
}

function decodeTag(value, mapping, field, file) {
  const tag = value?.tag;
  if (!tag || !mapping[tag]) {
    fail(`${file} contains unknown ${field} tag ${JSON.stringify(value)}`);
  }
  return mapping[tag];
}

function sha256(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

function stable(value) {
  if (Array.isArray(value)) {
    return value.map(stable);
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
}

function canonicalJson(value) {
  return `${JSON.stringify(stable(value), null, 2)}\n`;
}

function fail(message) {
  console.error(message);
  process.exit(1);
}
