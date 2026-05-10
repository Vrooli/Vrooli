#!/usr/bin/env node
import { createHash } from "node:crypto";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const scriptPath = fileURLToPath(import.meta.url);
const generatorPath = path.relative(path.resolve(path.dirname(scriptPath), "../.."), scriptPath);
const templateRoot = path.resolve(path.dirname(scriptPath), "../..");
const generatorVersion = 2;

const parsedArgs = parseArgs(process.argv.slice(2));
if (parsedArgs.help) {
  console.log("Usage: node tools/temporal-model/generate.mjs [--check] [--list] [--flow <flow-id>]");
  process.exit(0);
}

const flowContracts = discoverFlowContracts();
if (parsedArgs.list) {
  for (const contract of flowContracts) {
    console.log(contract.flowId);
  }
  process.exit(0);
}

const selectedContracts = parsedArgs.flow
  ? flowContracts.filter((contract) => contract.flowId === parsedArgs.flow)
  : flowContracts;
if (selectedContracts.length === 0) {
  fail(`unknown flow id ${parsedArgs.flow}`);
}

const quintVersion = run(["quint", "--version"], { quiet: true }).stdout.trim();
if (!quintVersion) {
  fail("quint --version returned an empty version");
}

try {
  let wrote = 0;
  for (const contract of selectedContracts) {
    validateContract(contract);
    const renderedModel = renderQuint(contract);
    const modelPath = path.join(templateRoot, contract.outputs.modelPath);
    if (parsedArgs.check) {
      assertFileFresh(contract.outputs.modelPath, renderedModel, contract.flowId);
    }
    const artifact = generateArtifact(contract, renderedModel, quintVersion, { writeModel: !parsedArgs.check });
    const artifactPath = path.join(templateRoot, contract.outputs.artifactPath);
    const nextArtifact = canonicalJson(artifact);

    if (parsedArgs.check) {
      assertFileFresh(contract.outputs.artifactPath, nextArtifact, contract.flowId);
      console.log(`fresh ${contract.flowId}`);
      continue;
    }

    mkdirSync(path.dirname(modelPath), { recursive: true });
    writeFileSync(modelPath, renderedModel);
    mkdirSync(path.dirname(artifactPath), { recursive: true });
    writeFileSync(artifactPath, nextArtifact);
    wrote += 1;
    console.log(`wrote ${contract.outputs.modelPath}`);
    console.log(`wrote ${contract.outputs.artifactPath}`);
  }

  if (!parsedArgs.check) {
    console.log(`generated ${wrote} temporal flow(s)`);
  }
} catch (error) {
  console.error(error.message);
  process.exit(1);
}

function parseArgs(args) {
  const parsed = { check: false, flow: "", help: false, list: false };
  for (let i = 0; i < args.length; i += 1) {
    const arg = args[i];
    if (arg === "--check") {
      parsed.check = true;
    } else if (arg === "--list") {
      parsed.list = true;
    } else if (arg === "--help") {
      parsed.help = true;
    } else if (arg === "--flow") {
      parsed.flow = args[i + 1] ?? "";
      i += 1;
      if (!parsed.flow) {
        fail("--flow requires a flow id");
      }
    } else {
      fail(`unknown argument ${arg}`);
    }
  }
  return parsed;
}

function discoverFlowContracts() {
  return findFiles(templateRoot, ".flow.json")
    .map((filePath) => loadContract(filePath))
    .sort((left, right) => left.flowId.localeCompare(right.flowId));
}

function findFiles(root, suffix) {
  const ignored = new Set([".git", "node_modules", "dist", "build", "coverage", "_apalache-out"]);
  const found = [];
  const visit = (dir) => {
    for (const entry of readdirSync(dir).sort()) {
      if (ignored.has(entry)) {
        continue;
      }
      const abs = path.join(dir, entry);
      const stats = statSync(abs);
      if (stats.isDirectory()) {
        visit(abs);
      } else if (entry.endsWith(suffix)) {
        found.push(abs);
      }
    }
  };
  visit(root);
  return found;
}

function loadContract(absPath) {
  let contract;
  try {
    contract = JSON.parse(readFileSync(absPath, "utf8"));
  } catch (error) {
    fail(`parse ${relativePath(absPath)}: ${error.message}`);
  }
  return { ...contract, contractPath: relativePath(absPath) };
}

function validateContract(contract) {
  const errors = [];
  validateAllowedKeys(errors, "contract", contract, [
    "schemaVersion",
    "flowId",
    "domain",
    "description",
    "model",
    "outputs",
    "states",
    "events",
    "transitions",
    "invariants",
    "traces",
    "runtime",
    "contractPath",
  ]);
  validateAllowedKeys(errors, "model", contract.model, ["module", "seed", "maxSteps", "traceCount", "verify"]);
  validateAllowedKeys(errors, "model.verify", contract.model?.verify, ["invariants"]);
  validateAllowedKeys(errors, "outputs", contract.outputs, ["modelPath", "artifactPath"]);
  const requireString = (pathName, value) => {
    if (typeof value !== "string" || value.trim() === "") {
      errors.push(`${pathName} is required`);
    }
  };
  if (contract.schemaVersion !== 1) {
    errors.push("schemaVersion must be 1");
  }
  requireString("flowId", contract.flowId);
  requireString("domain", contract.domain);
  requireString("description", contract.description);
  requireString("model.module", contract.model?.module);
  requireString("model.seed", contract.model?.seed);
  requireString("outputs.modelPath", contract.outputs?.modelPath);
  requireString("outputs.artifactPath", contract.outputs?.artifactPath);
  if (!Number.isInteger(contract.model?.maxSteps) || contract.model.maxSteps < 1) {
    errors.push("model.maxSteps must be a positive integer");
  }
  if (!Number.isInteger(contract.model?.traceCount) || contract.model.traceCount < 1) {
    errors.push("model.traceCount must be a positive integer");
  }
  if (!Array.isArray(contract.model?.verify?.invariants) || contract.model.verify.invariants.length === 0) {
    errors.push("model.verify.invariants must declare at least one invariant");
  }

  const states = validateNamedList(errors, "states", contract.states, ["id", "quint"], ["id", "quint", "initial", "terminal"]);
  const events = validateNamedList(errors, "events", contract.events, ["id", "quint"], ["id", "quint"]);
  const invariants = validateNamedList(
    errors,
    "invariants",
    contract.invariants,
    ["id", "quint"],
    ["id", "quint", "description", "expression"],
  );
  const stateIds = new Set(states.map((state) => state.id));
  const eventIds = new Set(events.map((event) => event.id));
  const invariantNames = new Set(invariants.map((invariant) => invariant.quint));
  const declaredInitial = states.filter((state) => state.initial);
  if (declaredInitial.length !== 1) {
    errors.push(`states must declare exactly one initial state, got ${declaredInitial.length}`);
  }
  for (const invariant of contract.model?.verify?.invariants ?? []) {
    if (!invariantNames.has(invariant)) {
      errors.push(`model.verify.invariants references unknown invariant ${invariant}`);
    }
  }

  if (!Array.isArray(contract.transitions) || contract.transitions.length === 0) {
    errors.push("transitions must not be empty");
  }
  const seenPairs = new Set();
  for (const [index, transition] of (contract.transitions ?? []).entries()) {
    validateAllowedKeys(errors, `transitions[${index}]`, transition, ["from", "event", "to", "wantError"]);
    if (!stateIds.has(transition.from)) {
      errors.push(`transitions[${index}].from references unknown state ${transition.from}`);
    }
    if (!stateIds.has(transition.to)) {
      errors.push(`transitions[${index}].to references unknown state ${transition.to}`);
    }
    if (!eventIds.has(transition.event)) {
      errors.push(`transitions[${index}].event references unknown event ${transition.event}`);
    }
    if (typeof transition.wantError !== "boolean") {
      errors.push(`transitions[${index}].wantError must be boolean`);
    }
    const pair = `${transition.from}\u0000${transition.event}`;
    if (seenPairs.has(pair)) {
      errors.push(`duplicate transition pair ${transition.from}/${transition.event}`);
    }
    seenPairs.add(pair);
  }
  for (const state of states) {
    for (const event of events) {
      if (!seenPairs.has(`${state.id}\u0000${event.id}`)) {
        errors.push(`missing transition pair ${state.id}/${event.id}`);
      }
    }
  }

  if (!Array.isArray(contract.traces) || contract.traces.length === 0) {
    errors.push("traces must not be empty");
  }
  for (const [traceIndex, trace] of (contract.traces ?? []).entries()) {
    validateAllowedKeys(errors, `traces[${traceIndex}]`, trace, ["name", "initial", "steps"]);
    if (!stateIds.has(trace.initial)) {
      errors.push(`traces[${traceIndex}].initial references unknown state ${trace.initial}`);
    }
    for (const [stepIndex, step] of (trace.steps ?? []).entries()) {
      validateAllowedKeys(errors, `traces[${traceIndex}].steps[${stepIndex}]`, step, ["event", "want", "wantError"]);
      if (!eventIds.has(step.event)) {
        errors.push(`traces[${traceIndex}].steps[${stepIndex}].event references unknown event ${step.event}`);
      }
      if (!stateIds.has(step.want)) {
        errors.push(`traces[${traceIndex}].steps[${stepIndex}].want references unknown state ${step.want}`);
      }
      if (typeof step.wantError !== "boolean") {
        errors.push(`traces[${traceIndex}].steps[${stepIndex}].wantError must be boolean`);
      }
    }
  }

  if (errors.length > 0) {
    fail(`invalid temporal flow contract ${contract.contractPath}:\n${formatErrors(errors)}`);
  }
}

function validateNamedList(errors, pathName, values, requiredKeys, allowedKeys) {
  if (!Array.isArray(values) || values.length === 0) {
    errors.push(`${pathName} must not be empty`);
    return [];
  }
  const seen = new Set();
  for (const [index, value] of values.entries()) {
    validateAllowedKeys(errors, `${pathName}[${index}]`, value, allowedKeys);
    for (const key of requiredKeys) {
      if (typeof value[key] !== "string" || value[key].trim() === "") {
        errors.push(`${pathName}[${index}].${key} is required`);
      }
    }
    if (seen.has(value.id)) {
      errors.push(`duplicate ${pathName} id ${value.id}`);
    }
    seen.add(value.id);
  }
  return values;
}

function validateAllowedKeys(errors, pathName, value, allowedKeys) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return;
  }
  const allowed = new Set(allowedKeys);
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) {
      errors.push(`${pathName}.${key} is not allowed`);
    }
  }
}

function generateArtifact(contract, renderedModel, quintVersion, options) {
  const modelAbs = path.join(templateRoot, contract.outputs.modelPath);
  const tempDir = mkdtempSync(path.join(tmpdir(), "react-vite-temporal-model-"));
  const itfPattern = path.join(tempDir, `${contract.flowId.replaceAll(".", "-")}_{seq}.itf.json`);
  const invariantNames = contract.model.verify.invariants;
  const commands = {
    typecheck: ["quint", "typecheck", contract.outputs.modelPath],
    test: ["quint", "test", contract.outputs.modelPath, "--seed", contract.model.seed],
    verify: [
      "quint",
      "verify",
      contract.outputs.modelPath,
      "--invariants",
      ...invariantNames,
      "--max-steps",
      String(contract.model.maxSteps),
    ],
    run: [
      "quint",
      "run",
      contract.outputs.modelPath,
      "--mbt",
      "--seed",
      contract.model.seed,
      "--max-samples",
      String(contract.model.traceCount),
      "--n-traces",
      String(contract.model.traceCount),
      "--max-steps",
      String(contract.model.maxSteps),
      "--out-itf",
      "<temp-itf-pattern>",
    ],
  };

  try {
    if (options.writeModel) {
      mkdirSync(path.dirname(modelAbs), { recursive: true });
      writeFileSync(modelAbs, renderedModel);
    }
    run(commands.typecheck);
    run(commands.test);
    run(commands.verify);
    run(commands.run.map((part) => (part === "<temp-itf-pattern>" ? itfPattern : part)));

    return stable({
      schemaVersion: 2,
      flowId: contract.flowId,
      source: {
        contractPath: contract.contractPath,
        contractSha256: fileSha256(path.join(templateRoot, contract.contractPath)),
        modelPath: contract.outputs.modelPath,
        modelSha256: sha256(renderedModel),
        generatorPath,
        generatorSha256: fileSha256(scriptPath),
        generatorVersion,
        quintVersion,
        verificationBackend: "apalache",
      },
      commands,
      states: contract.states.map(({ id }) => id),
      events: contract.events.map(({ id }) => id),
      transitions: contract.transitions.map(({ from, event, to, wantError }) => ({ from, event, to, wantError })),
      namedTraces: contract.traces.map((trace) => ({
        name: trace.name,
        initial: trace.initial,
        steps: trace.steps.map(({ event, want, wantError }) => ({ event, want, wantError })),
      })),
      generatedTraces: normalizeTraces(contract, tempDir),
      invariants: contract.model.verify.invariants,
      coverage: coverageSummary(contract),
      checks: {
        typechecked: true,
        tested: true,
        verified: true,
        generatedFromContract: true,
        generatedFromModel: true,
      },
    });
  } finally {
    rmSync(tempDir, { force: true, recursive: true });
    rmSync(path.join(templateRoot, "_apalache-out"), { force: true, recursive: true });
  }
}

function renderQuint(contract) {
  const states = contract.states;
  const events = contract.events;
  const initial = states.find((state) => state.initial);
  const stateTags = states.map((state) => state.quint);
  const eventTags = events.map((event) => event.quint);
  const transitionClauses = contract.transitions
    .filter((transition) => !transition.wantError && transition.from !== transition.to)
    .map((transition) => {
      const from = quintState(contract, transition.from);
      const event = quintEvent(contract, transition.event);
      const to = quintState(contract, transition.to);
      return `    else if (s == ${from} and e == ${event}) ${to}`;
    });
  const validClauses = contract.transitions
    .filter((transition) => !transition.wantError)
    .map((transition) => `(s == ${quintState(contract, transition.from)} and e == ${quintEvent(contract, transition.event)})`);
  const transitionAssertions = contract.transitions.map((transition, index) => {
    const from = quintState(contract, transition.from);
    const event = quintEvent(contract, transition.event);
    const to = quintState(contract, transition.to);
    const expectation = transition.wantError
      ? `not(isValid(${from}, ${event})) and nextStatus(${from}, ${event}) == ${to}`
      : `isValid(${from}, ${event}) and nextStatus(${from}, ${event}) == ${to}`;
    const prefix = index === 0 ? "    assert" : "      .then(assert";
    const suffix = index === 0 ? ")" : "))";
    return `${prefix}(${expectation}${suffix}`;
  });
  const terminalStates = states.filter((state) => state.terminal).map((state) => state.quint);
  const terminalClosure = terminalStates.length === 0
    ? "true"
    : terminalStates
      .map((state) => `(status != ${state} or (${eventTags.map((event) => `nextStatus(${state}, ${event}) == ${state}`).join(" and ")}))`)
      .join(" and\n    ");

  return `${generatedHeader(contract)}
module ${contract.model.module} {
  type Status = ${stateTags.join(" | ")}
  type Event = ${eventTags.join(" | ")}

  var status: Status
  var event: Event
  var rejected: bool

  pure def isValid(s: Status, e: Event): bool =
    ${validClauses.length > 0 ? validClauses.join(" or\n    ") : "false"}

  pure def nextStatus(s: Status, e: Event): Status =
${transitionClauses.length > 0 ? transitionClauses.join("\n").replace(/^    else if /, "    if ") : "    s"}
    else s

  action init = all {
    status' = ${initial.quint},
    event' = ${eventTags[0]},
    rejected' = false,
  }

  action apply(e: Event): bool = all {
    event' = e,
    status' = nextStatus(status, e),
    rejected' = not(isValid(status, e)),
  }

  action step = any {
${eventTags.map((event) => `    apply(${event}),`).join("\n")}
  }

  val TypeOK = status.in(Set(${stateTags.join(", ")})) and
    event.in(Set(${eventTags.join(", ")}))

  val TerminalClosure =
    ${terminalClosure}

  val IllegalTransitionsPreserveState =
    not(rejected) or nextStatus(status, event) == status

  val AllDeclaredTransitionsCovered = true
${contract.invariants
  .filter((invariant) => !["TypeOK", "TerminalClosure", "IllegalTransitionsPreserveState", "AllDeclaredTransitionsCovered"].includes(invariant.quint))
  .map((invariant) => `\n  val ${invariant.quint} =\n    ${invariant.expression ?? "true"}`)
  .join("")}

  run transitionTable = {
${transitionAssertions.join("\n")}
  }
}
`;
}

function generatedHeader(contract) {
  return `// Code generated by tools/temporal-model/generate.mjs from ${contract.contractPath}; DO NOT EDIT.\n`;
}

function normalizeTraces(contract, tempDir) {
  const files = readdirSync(tempDir).sort();
  const stateByQuint = new Map(contract.states.map((state) => [state.quint, state.id]));
  const eventByQuint = new Map(contract.events.map((event) => [event.quint, event.id]));

  return files.map((file, index) => {
    const raw = JSON.parse(readFileSync(path.join(tempDir, file), "utf8"));
    const states = raw.states ?? [];
    if (states.length === 0) {
      fail(`${file} contains no ITF states`);
    }

    const initial = decodeTag(states[0].status, stateByQuint, "status", file);
    const steps = states.slice(1).map((state) => ({
      event: decodeTag(state.event, eventByQuint, "event", file),
      want: decodeTag(state.status, stateByQuint, "status", file),
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
  if (!tag || !mapping.has(tag)) {
    fail(`${file} contains unknown ${field} tag ${JSON.stringify(value)}`);
  }
  return mapping.get(tag);
}

function coverageSummary(contract) {
  const transitionPairs = new Set(contract.transitions.map((transition) => `${transition.from}\u0000${transition.event}`));
  const traceStates = new Set();
  const traceEvents = new Set();
  for (const trace of contract.traces) {
    traceStates.add(trace.initial);
    for (const step of trace.steps) {
      traceEvents.add(step.event);
      traceStates.add(step.want);
    }
  }
  return {
    allStatesCovered: contract.states.every((state) => traceStates.has(state.id)),
    allEventsCovered: contract.events.every((event) => traceEvents.has(event.id)),
    allPairsCovered: contract.states.every((state) =>
      contract.events.every((event) => transitionPairs.has(`${state.id}\u0000${event.id}`))),
    terminalStatesChecked: contract.states.filter((state) => state.terminal).every((state) =>
      contract.events.every((event) => transitionPairs.has(`${state.id}\u0000${event.id}`))),
  };
}

function quintState(contract, stateId) {
  return contract.states.find((state) => state.id === stateId).quint;
}

function quintEvent(contract, eventId) {
  return contract.events.find((event) => event.id === eventId).quint;
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
    throw new Error([`command failed: ${rendered}`, stdout, stderr].filter(Boolean).join("\n"));
  }
  return result;
}

function assertFileFresh(filePath, next, flowId) {
  if (!existsSync(path.join(templateRoot, filePath))) {
    fail(`${filePath} is missing. Run node tools/temporal-model/generate.mjs --flow ${flowId}`);
  }
  const current = readFileSync(path.join(templateRoot, filePath), "utf8");
  if (current !== next) {
    fail(`${filePath} is stale. Run node tools/temporal-model/generate.mjs --flow ${flowId}`);
  }
}

function sha256(value) {
  return createHash("sha256").update(value).digest("hex");
}

function fileSha256(absPath) {
  return sha256(readFileSync(absPath));
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

function relativePath(absPath) {
  return path.relative(templateRoot, absPath).split(path.sep).join("/");
}

function formatErrors(errors) {
  return errors.map((error) => `  - ${error}`).join("\n");
}

function fail(message) {
  console.error(message);
  process.exit(1);
}
