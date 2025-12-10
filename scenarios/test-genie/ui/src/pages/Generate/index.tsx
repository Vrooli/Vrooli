import { useEffect, useMemo, useState, useCallback, useRef } from "react";
import { PhaseSelector } from "./PhaseSelector";
import { PromptEditor } from "./PromptEditor";
import { ActionButtons } from "./ActionButtons";
import { PresetSelector } from "./PresetSelector";
import { ScenarioTargetDialog } from "./ScenarioTargetDialog";
import { ActiveAgentsPanel } from "./ActiveAgentsPanel";
import { ConflictPreview } from "./ConflictPreview";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import { PHASES_FOR_GENERATION, PHASE_LABELS } from "../../lib/constants";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { AgentModel, SpawnAgentsResult, ConflictDetail, fetchAgentModels, spawnAgents, checkScopeConflicts, fetchAppConfig, validatePromptPaths, type AppConfig, type ValidatePathsResponse } from "../../lib/api";

const MAX_PROMPTS = 12;
const RECENT_MODEL_STORAGE_KEY = "test-genie-recent-agent-models";
const SESSION_SPAWNS_KEY = "test-genie-session-spawns";

/**
 * Session-level spawn tracking to prevent accidentally re-spawning agents
 * for the same scope while they're still running or recently completed.
 */
interface SessionSpawnRecord {
  scenario: string;
  scope: string[];
  phases: string[];
  agentIds: string[];
  spawnedAt: string;
  status: "active" | "completed" | "failed";
}

function loadSessionSpawns(): SessionSpawnRecord[] {
  try {
    const raw = sessionStorage.getItem(SESSION_SPAWNS_KEY);
    if (raw) {
      return JSON.parse(raw) as SessionSpawnRecord[];
    }
  } catch {
    // Ignore storage errors
  }
  return [];
}

function saveSessionSpawns(spawns: SessionSpawnRecord[]): void {
  try {
    sessionStorage.setItem(SESSION_SPAWNS_KEY, JSON.stringify(spawns));
  } catch {
    // Ignore storage errors
  }
}

/**
 * Check if there are any session-level conflicts with the requested scope.
 * Returns the conflicting records if any exist.
 */
function checkSessionConflicts(
  scenario: string,
  scope: string[],
  spawns: SessionSpawnRecord[]
): SessionSpawnRecord[] {
  const activeSpawns = spawns.filter(
    (s) => s.status === "active" && s.scenario === scenario
  );

  if (activeSpawns.length === 0) return [];

  // If no scope specified (entire scenario), any active spawn for that scenario is a conflict
  if (scope.length === 0) {
    return activeSpawns;
  }

  // Check for path overlap
  return activeSpawns.filter((spawn) => {
    // If the previous spawn had no scope (entire scenario), it conflicts with everything
    if (spawn.scope.length === 0) return true;

    // Check for path overlap between scopes
    return scope.some((requestedPath) =>
      spawn.scope.some(
        (existingPath) =>
          requestedPath === existingPath ||
          requestedPath.startsWith(existingPath + "/") ||
          existingPath.startsWith(requestedPath + "/")
      )
    );
  });
}

type PromptCombination = {
  id: string;
  index: number;
  total: number;
  phases: string[];
  targets: string[];
  preamble: string;        // Immutable safety preamble (server-generated)
  body: string;            // Editable task body
  defaultBody: string;     // Default body for reset
  prompt: string;          // Full prompt (preamble + body) for display/copy
  defaultPrompt: string;   // Full default prompt for comparison
  label: string;
};

const comboId = (targets: string[], phases: string[]) =>
  `t:${targets.length > 0 ? targets.join("|") : "all"}|p:${phases.join("|")}`;

/**
 * Build the immutable safety preamble that enforces constraints.
 * This section CANNOT be edited by users - it's enforced at the system level.
 */
function buildSafetyPreamble(
  scenarioName: string,
  targetPaths: string[],
  repoRoot: string
): string {
  if (!scenarioName || !repoRoot) {
    return "";
  }

  const scenarioPath = `${repoRoot}/scenarios/${scenarioName}`;
  const hasTargets = targetPaths.length > 0;
  const absoluteTargets = hasTargets
    ? targetPaths.map((target) => `${scenarioPath}/${target}`)
    : [];

  const scopeDescription = hasTargets
    ? `Allowed scope: ${absoluteTargets.join(", ")}`
    : "Allowed scope: entire scenario directory";

  return `## SECURITY CONSTRAINTS (enforced by system - cannot be modified)

**Working directory:** ${scenarioPath}
**${scopeDescription}**

You MUST NOT:
- Access files outside ${scenarioPath}
- Execute destructive commands (rm -rf, git checkout --force, sudo, chmod 777, etc.)
- Modify system configurations, dependencies, or package files
- Delete or weaken existing tests without explicit rationale in comments
- Run commands that could affect other scenarios or system state

These constraints are enforced at the tool level. Violations will be blocked.

---

`;
}

/**
 * Build the editable task body that describes what the agent should do.
 * Users can customize this section while the preamble remains immutable.
 */
function buildTaskBody(
  scenarioName: string,
  selectedPhases: string[],
  preset: string | null,
  targetPaths: string[],
  repoRoot: string
): string {
  if (!scenarioName || selectedPhases.length === 0 || !repoRoot) {
    return "";
  }

  const phaseLabels = selectedPhases
    .map((p) => PHASE_LABELS[p] ?? p)
    .join(", ");
  const hasTargets = targetPaths.length > 0;

  let context = "";
  if (preset === "bootstrap") {
    context = `This is a new scenario that needs initial test coverage. Focus on establishing a solid foundation of tests.`;
  } else if (preset === "coverage") {
    context = `The scenario already has some tests. Focus on adding coverage for specific features or edge cases.`;
  } else if (preset === "fix-failing") {
    context = `Some tests are failing. Focus on understanding the failures and generating fixes or improved test cases.`;
  }

  const scenarioPath = `${repoRoot}/scenarios/${scenarioName}`;
  const scenarioDocsPath = `${scenarioPath}/docs`;
  const prdPath = `${scenarioPath}/PRD.md`;
  const requirementsPath = `${scenarioPath}/requirements`;
  const testGeniePath = `${repoRoot}/scenarios/test-genie`;
  const generalTestingDoc = `${testGeniePath}/docs/guides/test-generation.md`;

  const targetedRun = hasTargets
    ? `test-genie run-tests ${scenarioName} --type phased ${targetPaths
        .map((t) => `--path ${t}`)
        .join(" ")}`
    : `test-genie run-tests ${scenarioName} --type phased`;
  const phaseDocs = selectedPhases
    .map((phase) => PHASES_FOR_GENERATION.find((p) => p.key === phase))
    .filter(Boolean)
    .map((phase) => `${testGeniePath}${phase!.docsPath}`);

  const phaseDocsList =
    phaseDocs.length > 0 ? phaseDocs.map((doc) => `- ${doc}`).join("\n") : "- (no phase docs selected)";

  const absoluteTargets = hasTargets
    ? targetPaths.map((target) => `${scenarioPath}/${target}`)
    : [];

  return `## Task: Generate tests for the "${scenarioName}" scenario

**Test phases:** ${phaseLabels}

${context ? `**Preset context:** ${context}\n\n` : ""}**Reference paths:**
- Scenario root: ${scenarioPath}
- PRD: ${prdPath}
- Requirements: ${requirementsPath}
- Scenario docs (if present): ${scenarioDocsPath}

**Docs to review first:**
- General test generation guide: ${generalTestingDoc}
- Phase docs for selected phases:
${phaseDocsList}

${hasTargets ? `**Targeted scope:**\n${absoluteTargets.map((p) => `- ${p}`).join("\n")}\n\n**Focus:** Add or enhance tests only for the components/files above.` : ""}

## Guidelines

**Best practices:**
- Align to PRD/requirements and existing coding/testing conventions in the scenario
- Make tests deterministic, isolated, and idempotent (no hidden state, fixed seeds, stable selectors)
- Add clear assertions and negative cases; keep names descriptive
- Use existing helpers/fixtures; prefer real interfaces for integration, limited mocking only where already used
- Mark requirement coverage if applicable (e.g., \`[REQ:ID]\`) and keep coverage neutral or improved
- Call out any assumptions and uncertain areas explicitly

**Avoid:**
- Flaky behavior (timing sleeps, random data without seeds, network calls to external services)
- Unnecessary mocking when real implementations are available
- Over-complex test setups that obscure intent

## Generation steps

1) Read PRD, requirements, and existing tests to mirror patterns
2) For each selected phase, target meaningful coverage:
   - Unit: small, fast, local; no network/filesystem unless already mocked
   - Integration: exercise real component boundaries; minimal mocking; preserve setup/teardown hygiene
   - Playbooks (E2E): stable selectors, retries around navigation, capture screenshots/logs on failure hooks
   - Business: assertions tied to requirements/PRD acceptance criteria
3) Create or update tests under the scenario using existing structure and naming${hasTargets ? "; limit changes to the targeted paths above" : ""}
4) Keep changes idempotent and documented (comments only where intent is non-obvious)

## Validation (run after generation)

- Execute via test-genie with the selected phases:
  test-genie execute ${scenarioName} --phases ${selectedPhases.join(",")} --sync
- Fast local check (phased runner):
  ${targetedRun}
- If preset implies broader coverage, also run:
  test-genie execute ${scenarioName} --preset comprehensive --sync
- When targets are set, re-run unit commands focused on those files after generation to confirm determinism.
- If any failures occur, include logs/output snippets and suggested fixes.

## Expected output

When you complete your work, provide a summary in the following JSON format wrapped in a \`\`\`json code block:

\`\`\`json
{
  "status": "success" | "partial" | "failed",
  "summary": "Brief description of what was accomplished",
  "filesChanged": [
    {
      "path": "relative/path/to/file.ts",
      "action": "created" | "modified" | "deleted",
      "rationale": "Why this change was made"
    }
  ],
  "testsAdded": {
    "count": 5,
    "byPhase": {
      "unit": 3,
      "integration": 2
    }
  },
  "commandsRun": [
    {
      "command": "test-genie execute ...",
      "result": "passed" | "failed",
      "output": "Brief output or error message"
    }
  ],
  "coverageImpact": {
    "before": 75.5,
    "after": 82.3,
    "delta": 6.8
  },
  "blockers": [
    {
      "type": "missing_dependency" | "unclear_requirement" | "test_failure" | "other",
      "description": "What blocked progress",
      "suggestedResolution": "How it might be resolved"
    }
  ],
  "assumptions": [
    "Any assumptions made during implementation"
  ],
  "nextSteps": [
    "Suggested follow-up actions"
  ]
}
\`\`\`

This structured output helps track progress and identify patterns across multiple agent runs.

Please generate comprehensive ${phaseLabels.toLowerCase()} for this scenario.`;
}

/**
 * Combine preamble and body into the full prompt.
 */
function buildFullPrompt(preamble: string, body: string): string {
  if (!preamble || !body) return "";
  return preamble + body;
}

export function GeneratePage() {
  const { scenarioDirectoryEntries } = useScenarios();
  const { focusScenario, setFocusScenario } = useUIStore();

  const [selectedPhases, setSelectedPhases] = useState<string[]>(["unit", "integration"]);
  const [selectedPreset, setSelectedPreset] = useState<string | null>(null);
  const [customPrompts, setCustomPrompts] = useState<Record<string, string>>({});
  const [targetPaths, setTargetPaths] = useState<string[]>([]);
  const [splitTargets, setSplitTargets] = useState(false);
  const [splitPhases, setSplitPhases] = useState(false);
  const [activePromptId, setActivePromptId] = useState<string | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [agentModel, setAgentModel] = useState("");
  const [recentModels, setRecentModels] = useState<string[]>([]);
  const [modelOptions, setModelOptions] = useState<AgentModel[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [modelsError, setModelsError] = useState<string | null>(null);
  const [spawnConcurrency, setSpawnConcurrency] = useState(3);
  const [maxTurns, setMaxTurns] = useState(12);
  const [timeoutSeconds, setTimeoutSeconds] = useState(300);
  const [allowedTools, setAllowedTools] = useState("read,edit,write,glob,grep");
  const [skipPermissions, setSkipPermissions] = useState(false);
  const [spawnBusy, setSpawnBusy] = useState(false);
  const [spawnStatus, setSpawnStatus] = useState<string | null>(null);
  const [spawnResults, setSpawnResults] = useState<SpawnAgentsResult[] | null>(null);
  const [scopeConflicts, setScopeConflicts] = useState<ConflictDetail[]>([]);
  const [conflictCheckPending, setConflictCheckPending] = useState(false);
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [pathValidation, setPathValidation] = useState<ValidatePathsResponse | null>(null);
  const conflictCheckTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Session-level spawn tracking (persists across page reloads within session)
  const [sessionSpawns, setSessionSpawns] = useState<SessionSpawnRecord[]>(() => loadSessionSpawns());
  const [sessionConflicts, setSessionConflicts] = useState<SessionSpawnRecord[]>([]);

  const hasTargets = targetPaths.length > 0;

  const handleTogglePhase = (phase: string) => {
    if (hasTargets) return; // When targeting paths, phases are locked to Unit
    setSelectedPhases((prev) =>
      prev.includes(phase)
        ? prev.filter((p) => p !== phase)
        : [...prev, phase]
    );
    setCustomPrompts({});
  };

  const handlePresetSelect = (preset: string) => {
    setSelectedPreset((prev) => (prev === preset ? null : preset));
    setCustomPrompts({});
  };

  // Load app config on mount (includes repoRoot)
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const config = await fetchAppConfig();
        setAppConfig(config);
      } catch (err) {
        console.error("Failed to load app config:", err);
        // Fallback to a sensible default if config fails
        setAppConfig({
          repoRoot: "/home/user/Vrooli",
          testGeniePath: "/home/user/Vrooli/scenarios/test-genie",
          testGenieCLI: "test-genie",
          scenariosPath: "/home/user/Vrooli/scenarios",
          timestamp: new Date().toISOString(),
          securityModel: "allowlist",
          directoryScoping: true,
          pathValidation: true,
          bashAllowlistOnly: true,
        });
      } finally {
        setConfigLoading(false);
      }
    };
    loadConfig();
  }, []);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(RECENT_MODEL_STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
          setRecentModels(parsed.filter((v) => typeof v === "string"));
        }
      }
    } catch {
      // ignore storage read errors
    }
  }, []);

  useEffect(() => {
    const loadModels = async () => {
      setModelsLoading(true);
      setModelsError(null);
      try {
        const models = await fetchAgentModels("openrouter");
        setModelOptions(models);
        const preferred = recentModels.find((m) => models.some((opt) => opt.id === m)) || models[0]?.id || "";
        setAgentModel(preferred);
      } catch (err) {
        setModelsError(err instanceof Error ? err.message : "Failed to load models");
      } finally {
        setModelsLoading(false);
      }
    };
    loadModels();
  }, [recentModels]);

  // Debounced conflict check when scenario or scope changes
  const checkForConflicts = useCallback(async () => {
    if (!focusScenario) {
      setScopeConflicts([]);
      return;
    }
    setConflictCheckPending(true);
    try {
      const result = await checkScopeConflicts(focusScenario, targetPaths);
      setScopeConflicts(result.conflicts);
    } catch (err) {
      console.error("Failed to check conflicts:", err);
      setScopeConflicts([]);
    } finally {
      setConflictCheckPending(false);
    }
  }, [focusScenario, targetPaths]);

  useEffect(() => {
    // Clear any pending timeout
    if (conflictCheckTimeout.current) {
      clearTimeout(conflictCheckTimeout.current);
    }

    // Debounce the conflict check by 300ms
    conflictCheckTimeout.current = setTimeout(() => {
      checkForConflicts();
    }, 300);

    return () => {
      if (conflictCheckTimeout.current) {
        clearTimeout(conflictCheckTimeout.current);
      }
    };
  }, [checkForConflicts]);

  // Check for session-level conflicts when scenario/scope changes
  useEffect(() => {
    if (!focusScenario) {
      setSessionConflicts([]);
      return;
    }
    const conflicts = checkSessionConflicts(focusScenario, targetPaths, sessionSpawns);
    setSessionConflicts(conflicts);
  }, [focusScenario, targetPaths, sessionSpawns]);

  // Update session spawns when agents complete (via WebSocket or polling)
  const updateSessionSpawnStatus = useCallback((agentId: string, status: "completed" | "failed") => {
    setSessionSpawns((prev) => {
      const updated = prev.map((spawn) => {
        if (spawn.agentIds.includes(agentId) && spawn.status === "active") {
          // Check if all agents in this spawn are done
          const allDone = spawn.agentIds.every((id) => id === agentId);
          if (allDone) {
            return { ...spawn, status };
          }
        }
        return spawn;
      });
      saveSessionSpawns(updated);
      return updated;
    });
  }, []);

  // Clear completed/failed session spawns older than 30 minutes
  useEffect(() => {
    const cleanup = () => {
      const thirtyMinutesAgo = Date.now() - 30 * 60 * 1000;
      setSessionSpawns((prev) => {
        const filtered = prev.filter((spawn) => {
          if (spawn.status === "active") return true;
          const spawnTime = new Date(spawn.spawnedAt).getTime();
          return spawnTime > thirtyMinutesAgo;
        });
        if (filtered.length !== prev.length) {
          saveSessionSpawns(filtered);
        }
        return filtered;
      });
    };

    // Run cleanup on mount and every 5 minutes
    cleanup();
    const interval = setInterval(cleanup, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  const saveRecentModel = (model: string) => {
    if (!model) return;
    setRecentModels((prev) => {
      const next = [model, ...prev.filter((m) => m !== model)].slice(0, 5);
      try {
        localStorage.setItem(RECENT_MODEL_STORAGE_KEY, JSON.stringify(next));
      } catch {
        // ignore storage write errors
      }
      return next;
    });
  };

  const handleScopeSave = (scenario: string, paths: string[]) => {
    setFocusScenario(scenario);
    setTargetPaths(paths);
    setCustomPrompts({});
    if (paths.length > 0) {
      setSelectedPhases(["unit"]);
    } else {
      setSelectedPhases(["unit", "integration"]);
    }
  };

  const targetGroups = useMemo(() => {
    if (splitTargets && targetPaths.length > 1) {
      return targetPaths.map((path) => [path]);
    }
    return [targetPaths];
  }, [splitTargets, targetPaths]);

  const phaseGroups = useMemo(() => {
    if (splitPhases && selectedPhases.length > 1) {
      return selectedPhases.map((phase) => [phase]);
    }
    return [selectedPhases];
  }, [splitPhases, selectedPhases]);

  const promptCombos = useMemo(() => {
    const combos: Array<{ phases: string[]; targets: string[] }> = [];
    targetGroups.forEach((targets) => {
      phaseGroups.forEach((phases) => combos.push({ phases, targets }));
    });
    return combos;
  }, [targetGroups, phaseGroups]);

  const totalCombos = promptCombos.length;
  const cappedCombos = promptCombos.slice(0, MAX_PROMPTS);
  const isCapped = totalCombos > MAX_PROMPTS;

  const promptItems: PromptCombination[] = useMemo(() => {
    const repoRoot = appConfig?.repoRoot ?? "";
    return cappedCombos.map((combo, index) => {
      const id = comboId(combo.targets, combo.phases);
      // Preamble is immutable - computed from scenario/targets only
      const preamble = buildSafetyPreamble(focusScenario, combo.targets, repoRoot);
      // Body is editable - contains task details
      const defaultBody = buildTaskBody(focusScenario, combo.phases, selectedPreset, combo.targets, repoRoot);
      const body = customPrompts[id] ?? defaultBody;
      // Full prompt combines both for display/copy
      const prompt = buildFullPrompt(preamble, body);
      const defaultPrompt = buildFullPrompt(preamble, defaultBody);
      const targetLabel =
        combo.targets.length === 0 ? "All paths" : combo.targets.join(", ");
      const phaseLabel = combo.phases.map((p) => PHASE_LABELS[p] ?? p).join(", ");
      return {
        id,
        index,
        total: cappedCombos.length,
        phases: combo.phases,
        targets: combo.targets,
        preamble,
        body,
        defaultBody,
        prompt,
        defaultPrompt,
        label: `Prompt ${index + 1} • ${targetLabel} • ${phaseLabel}`
      };
    });
  }, [cappedCombos, customPrompts, focusScenario, selectedPreset, appConfig]);

  useEffect(() => {
    if (!activePromptId && promptItems.length > 0) {
      setActivePromptId(promptItems[0].id);
      return;
    }
    if (activePromptId && promptItems.every((item) => item.id !== activePromptId)) {
      setActivePromptId(promptItems[0]?.id ?? null);
    }
  }, [activePromptId, promptItems]);

  const currentPromptItem = promptItems.find((item) => item.id === activePromptId) ?? promptItems[0];

  const handleBodyChange = (id: string, value: string, defaultBody: string) => {
    setCustomPrompts((prev) => {
      const next = { ...prev };
      if (value === defaultBody) {
        delete next[id];
      } else {
        next[id] = value;
      }
      return next;
    });
  };

  const isDisabled = !focusScenario || selectedPhases.length === 0 || !currentPromptItem;
  const promptCount = promptItems.length;
  const safeDisplayConcurrency = Math.max(
    1,
    Math.min(Number.isFinite(spawnConcurrency) ? spawnConcurrency : 1, Math.max(promptCount, 1))
  );
  const spawnDisabled =
    isDisabled || promptCount === 0 || !agentModel || modelsLoading || Boolean(modelsError);

  const handleSpawnAll = async () => {
    if (spawnDisabled || promptCount === 0) return;
    const safeConcurrency = Number.isFinite(spawnConcurrency) ? spawnConcurrency : 1;
    const safeMaxTurns = Number.isFinite(maxTurns) ? maxTurns : 0;
    const safeTimeout = Number.isFinite(timeoutSeconds) ? timeoutSeconds : 0;
    setSpawnBusy(true);
    setSpawnStatus("Checking for scope conflicts...");
    setSpawnResults(null);
    setScopeConflicts([]);

    try {
      // Check for session-level conflicts first (local tracking)
      if (focusScenario) {
        const localConflicts = checkSessionConflicts(focusScenario, targetPaths, sessionSpawns);
        if (localConflicts.length > 0) {
          setSessionConflicts(localConflicts);
          const scopeDescriptions = localConflicts.map((c) =>
            c.scope.length > 0 ? c.scope.join(", ") : "entire scenario"
          );
          setSpawnStatus(
            `Session conflict: You already spawned agents for overlapping scope in this session (${scopeDescriptions.join("; ")}). ` +
            `Wait for them to complete, stop them, or clear session history.`
          );
          setSpawnBusy(false);
          return;
        }
      }

      // Check for scope conflicts before spawning (server-side)
      if (focusScenario) {
        const conflictCheck = await checkScopeConflicts(focusScenario, targetPaths);
        if (conflictCheck.hasConflicts) {
          setScopeConflicts(conflictCheck.conflicts);
          setSpawnStatus(`Scope conflict: ${conflictCheck.conflicts.length} agent(s) already working on overlapping paths. Wait for them to complete or stop them first.`);
          setSpawnBusy(false);
          return;
        }
      }

      // Pre-flight validation: check that referenced paths exist
      setSpawnStatus("Validating prompt paths...");
      if (focusScenario) {
        try {
          const validation = await validatePromptPaths(focusScenario, selectedPhases);
          setPathValidation(validation);
          if (!validation.valid) {
            // Required paths missing - block spawn
            setSpawnStatus(`Path validation failed: ${validation.errors.join("; ")}`);
            setSpawnBusy(false);
            return;
          }
          // Show warnings but continue (they're non-blocking)
          if (validation.warnings.length > 0) {
            console.warn("Path validation warnings:", validation.warnings);
          }
        } catch (err) {
          // Log but don't block on validation errors
          console.warn("Path validation check failed:", err);
        }
      }

      setSpawnStatus("Spawning agents via OpenCode/OpenRouter...");
      const allowed = allowedTools
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);
      // Get the preamble from the first prompt item (same for all prompts in this batch)
      const preamble = promptItems[0]?.preamble;
      const res = await spawnAgents({
        prompts: promptItems.map((p) => p.prompt),
        preamble: preamble || undefined,
        model: agentModel,
        concurrency: Math.max(1, Math.min(safeConcurrency, promptCount)),
        maxTurns: safeMaxTurns > 0 ? safeMaxTurns : undefined,
        timeoutSeconds: safeTimeout > 0 ? safeTimeout : undefined,
        allowedTools: allowed.length ? allowed : undefined,
        skipPermissions,
        scenario: focusScenario || undefined,
        scope: targetPaths.length > 0 ? targetPaths : undefined,
        phases: selectedPhases.length > 0 ? selectedPhases : undefined
      });

      // Check for scope conflicts in the response (just agent IDs)
      // If there are conflicts, re-fetch detailed conflict info
      if (res.scopeConflicts && res.scopeConflicts.length > 0) {
        setSpawnStatus(`Scope conflict: ${res.scopeConflicts.length} agent(s) already working on overlapping paths.`);
        // Refresh detailed conflict info
        checkForConflicts();
        return;
      }

      setSpawnResults(res.items ?? []);
      saveRecentModel(agentModel);

      // Record this spawn in session storage for conflict tracking
      const agentIds = res.items
        .filter((item) => item.agentId)
        .map((item) => item.agentId as string);
      if (agentIds.length > 0 && focusScenario) {
        const newSpawn: SessionSpawnRecord = {
          scenario: focusScenario,
          scope: targetPaths,
          phases: selectedPhases,
          agentIds,
          spawnedAt: new Date().toISOString(),
          status: "active",
        };
        setSessionSpawns((prev) => {
          const updated = [...prev, newSpawn];
          saveSessionSpawns(updated);
          return updated;
        });
      }

      const cappedNote = res.capped ? " (first batch capped; narrow scope to spawn all)" : "";
      setSpawnStatus(`Spawned ${res.items.length} agent${res.items.length === 1 ? "" : "s"}${cappedNote}.`);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : "Failed to spawn agents";
      setSpawnStatus(errMsg);
    } finally {
      setSpawnBusy(false);
    }
  };

  const handleConflictDetected = (_conflictingAgentIds: string[]) => {
    // Refresh detailed conflict info when conflicts are detected
    checkForConflicts();
  };

  const handleConflictAgentStopped = (agentId: string) => {
    // Remove conflicts related to the stopped agent and refresh
    setScopeConflicts(prev => prev.filter(c => c.lockedBy.agentId !== agentId));
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <section className="rounded-2xl border border-white/10 bg-gradient-to-r from-purple-500/10 via-transparent to-cyan-500/10 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test Generation</p>
        <h1 className="mt-2 text-2xl font-semibold">Generate test prompts</h1>
        <p className="mt-2 text-sm text-slate-300">
          Build prompts for AI-powered test generation. Select phases, customize the prompt,
          and copy it for use with your preferred AI assistant.
        </p>
      </section>

      {/* Scenario & Targets */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Target</p>
            <h3 className="mt-2 text-lg font-semibold">Scenario & scope</h3>
            <p className="mt-2 text-sm text-slate-300">
              Choose a scenario and optionally focus on specific api/ui folders or files for targeted generation.
            </p>
          </div>
          <Button onClick={() => setIsDialogOpen(true)} data-testid={selectors.generate.scopeButton}>
            {focusScenario ? "Update selection" : "Select scenario"}
          </Button>
        </div>

        <div className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Current</p>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-white">
            <span className="rounded-full border border-white/15 bg-white/5 px-3 py-1">
              Scenario: {focusScenario || "Not selected"}
            </span>
            <span className="rounded-full border border-white/15 bg-white/5 px-3 py-1">
              Targets: {targetPaths.length > 0 ? `${targetPaths.length} selected` : "All paths"}
            </span>
          </div>
          {targetPaths.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-2">
              {targetPaths.slice(0, 6).map((path) => (
                <span key={path} className="rounded-full border border-cyan-400/40 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-100">
                  {path}
                </span>
              ))}
              {targetPaths.length > 6 && (
                <span className="text-xs text-slate-400">+{targetPaths.length - 6} more</span>
              )}
            </div>
          )}
        </div>
        <div className="mt-4 flex flex-col gap-2">
          <label className={cn(
            "flex items-start gap-3 rounded-lg border px-3 py-2 text-sm",
            splitTargets ? "border-cyan-400/60 bg-cyan-400/5" : "border-white/10 bg-white/[0.02]"
          )}>
            <input
              type="checkbox"
              className="mt-1 h-4 w-4"
              checked={splitTargets}
              onChange={(e) => setSplitTargets(e.target.checked)}
            />
            <div>
              <p className="font-semibold text-white">Split prompts by target path</p>
              <p className="text-xs text-slate-400">
                Generates one prompt per selected folder/file. Requires at least two targets.
              </p>
            </div>
          </label>
          {targetPaths.length < 2 && splitTargets && (
            <p className="text-xs text-amber-300">Select two or more targets to split prompts.</p>
          )}
        </div>
      </section>

      {/* Preset Templates */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <PresetSelector
          selectedPreset={selectedPreset}
          onSelectPreset={handlePresetSelect}
        />
      </section>

      {/* Phase Selection */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <PhaseSelector
          selectedPhases={selectedPhases}
          lockToUnit={hasTargets}
          onTogglePhase={handleTogglePhase}
        />
        <div className="mt-4 flex flex-col gap-2">
          <label className={cn(
            "flex items-start gap-3 rounded-lg border px-3 py-2 text-sm",
            splitPhases ? "border-cyan-400/60 bg-cyan-400/5" : "border-white/10 bg-white/[0.02]"
          )}>
            <input
              type="checkbox"
              className="mt-1 h-4 w-4"
              checked={splitPhases}
              onChange={(e) => setSplitPhases(e.target.checked)}
            />
            <div>
              <p className="font-semibold text-white">Split prompts by phase</p>
              <p className="text-xs text-slate-400">
                Generates one prompt per selected phase. Keep integration/business combined if they share context.
              </p>
            </div>
          </label>
          {selectedPhases.length < 2 && splitPhases && (
            <p className="text-xs text-amber-300">Select two or more phases to split prompts.</p>
          )}
        </div>
      </section>

      {/* Prompt Preview/Editor */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Prompt bundle</p>
            <h3 className="mt-2 text-lg font-semibold">
              Generated prompts {promptCount > 1 && `(${promptCount})`}
            </h3>
            {isCapped && (
              <p className="text-xs text-amber-300">
                Showing first {MAX_PROMPTS} prompts (capped). Narrow targets/phases to see all {totalCombos}.
              </p>
            )}
          </div>
          {promptCount > 1 && (
            <div className="flex items-center gap-2" data-testid={selectors.generate.promptNavigator}>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  if (!currentPromptItem) return;
                  const prevIndex = (currentPromptItem.index - 1 + promptCount) % promptCount;
                  setActivePromptId(promptItems[prevIndex].id);
                }}
              >
                Prev
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  if (!currentPromptItem) return;
                  const nextIndex = (currentPromptItem.index + 1) % promptCount;
                  setActivePromptId(promptItems[nextIndex].id);
                }}
              >
                Next
              </Button>
            </div>
          )}
        </div>

        {promptCount > 1 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {promptItems.map((item) => (
              <button
                key={item.id}
                type="button"
                className={cn(
                  "rounded-full border px-3 py-1 text-xs transition",
                  activePromptId === item.id
                    ? "border-cyan-400 bg-cyan-400/10 text-white"
                    : "border-white/10 bg-white/[0.02] text-slate-300 hover:border-white/30"
                )}
                onClick={() => setActivePromptId(item.id)}
                data-testid={selectors.generate.promptNavItem}
              >
                {item.label}
              </button>
            ))}
          </div>
        )}

        <div className="mt-4">
          <PromptEditor
            preamble={currentPromptItem?.preamble ?? ""}
            body={currentPromptItem?.body ?? ""}
            defaultBody={currentPromptItem?.defaultBody ?? ""}
            onBodyChange={(value) =>
              currentPromptItem && handleBodyChange(currentPromptItem.id, value, currentPromptItem.defaultBody)
            }
            fullPrompt={currentPromptItem?.prompt ?? ""}
            titleSuffix={
              promptCount > 1 && currentPromptItem
                ? `Prompt ${currentPromptItem.index + 1} of ${promptCount}`
                : undefined
            }
            summary={currentPromptItem?.label}
          />
        </div>
      </section>

      {/* Agent spawn settings */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Agent spawn</p>
        <h3 className="mt-2 text-lg font-semibold">Run prompts via OpenCode (OpenRouter)</h3>
        <p className="mt-2 text-sm text-slate-300">
          Spawn agents using the opencode resource (auto-wired to your OpenRouter API key). Configure the model and limits, then
          dispatch the current prompt bundle.
        </p>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Model</label>
            <select
              className="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
              value={agentModel}
              onChange={(e) => setAgentModel(e.target.value)}
              disabled={modelsLoading || modelOptions.length === 0}
            >
              {modelsLoading && <option>Loading models…</option>}
              {!modelsLoading && modelOptions.length === 0 && <option>No models available</option>}
              {!modelsLoading &&
                modelOptions.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.displayName || m.name || m.id} {m.provider ? `(${m.provider})` : ""}
                  </option>
                ))}
            </select>
            {modelsError && <p className="text-xs text-rose-300">Model load failed: {modelsError}</p>}
            {recentModels.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {recentModels.map((m) => (
                  <button
                    key={m}
                    type="button"
                    className="rounded-full border border-white/15 bg-white/[0.03] px-3 py-1 text-xs text-slate-200 hover:border-cyan-400"
                    onClick={() => setAgentModel(m)}
                  >
                    Recent: {m}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Concurrency</label>
              <div className="mt-1 flex items-center gap-3">
                <input
                  type="range"
                  min={1}
                  max={10}
                  value={spawnConcurrency}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setSpawnConcurrency(Number.isFinite(next) ? next : 1);
                  }}
                  className="w-full accent-cyan-400"
                />
                <input
                  type="number"
                  min={1}
                  max={10}
                  value={spawnConcurrency}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setSpawnConcurrency(Number.isFinite(next) ? next : 1);
                  }}
                  className="w-20 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                />
              </div>
              <p className="text-[11px] text-slate-400 mt-1">Max parallel agent runs (cap 10). Effective: {safeDisplayConcurrency}.</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Max turns</label>
              <div className="mt-1 flex items-center gap-3">
                <input
                  type="range"
                  min={0}
                  max={24}
                  value={maxTurns}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setMaxTurns(Number.isFinite(next) ? next : 0);
                  }}
                  className="w-full accent-cyan-400"
                />
                <input
                  type="number"
                  min={0}
                  value={maxTurns}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setMaxTurns(Number.isFinite(next) ? next : 0);
                  }}
                  className="w-20 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                />
              </div>
              <p className="text-[11px] text-slate-400 mt-1">0 = unlimited (not recommended). Higher turns improve reliability but cost more.</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Timeout (sec)</label>
              <div className="mt-1 flex items-center gap-3">
                <input
                  type="range"
                  min={60}
                  max={900}
                  step={15}
                  value={timeoutSeconds}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setTimeoutSeconds(Number.isFinite(next) ? next : 60);
                  }}
                  className="w-full accent-cyan-400"
                />
                <input
                  type="number"
                  min={0}
                  value={timeoutSeconds}
                  onChange={(e) => {
                    const next = Number(e.target.value);
                    setTimeoutSeconds(Number.isFinite(next) ? next : 0);
                  }}
                  className="w-24 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                />
              </div>
              <p className="text-[11px] text-slate-400 mt-1">Abort if a request exceeds this duration. Set 0 to rely on model defaults.</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Allowed tools</label>
              <input
                type="text"
                value={allowedTools}
                onChange={(e) => setAllowedTools(e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                placeholder="read,edit,write,glob,grep"
              />
              <p className="mt-1 text-[11px] text-slate-400">
                Keep <code className="font-mono">read</code> + <code className="font-mono">edit</code> enabled so agents can inspect files; add scoped bash (e.g. <code className="font-mono">bash(pnpm test|go test|vitest *)</code>) when targeting test commands.
              </p>
              <p className="mt-1 text-[11px] text-emerald-400">
                Destructive commands (git checkout, rm -rf, sudo, etc.) are blocked for safety.
              </p>
            </div>
          </div>
        </div>

        <div className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4 text-sm text-slate-200">
          <p className="font-semibold text-white">Dispatch summary</p>
          <p className="mt-1 text-slate-300">
            {promptCount} prompt{promptCount === 1 ? "" : "s"} • model {agentModel || "not selected"} • concurrency{" "}
            {safeDisplayConcurrency}
          </p>
          <p className="text-[11px] text-slate-400">
            Lower-cost models are fine for exploration; raise max turns/timeout for reliability but expect higher spend.
          </p>
          {spawnStatus && <p className="mt-2 text-xs text-cyan-300">{spawnStatus}</p>}
          {spawnResults && spawnResults.length > 0 && (
            <div className="mt-3 space-y-2">
              {spawnResults.map((res) => (
                <div key={res.promptIndex} className="rounded-lg border border-white/10 bg-white/[0.02] p-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-white">Prompt {res.promptIndex + 1}</span>
                    <span
                      className={cn(
                        "rounded-full px-2 py-1 text-[11px]",
                        res.status === "completed"
                          ? "bg-emerald-500/10 text-emerald-200 border border-emerald-400/40"
                          : res.status === "timeout"
                          ? "bg-amber-500/10 text-amber-200 border border-amber-400/40"
                          : "bg-rose-500/10 text-rose-200 border border-rose-400/40"
                      )}
                    >
                      {res.status}
                    </span>
                  </div>
                  {res.sessionId && <p className="mt-1 text-slate-300">Session: {res.sessionId}</p>}
                  {res.error && <p className="mt-1 text-rose-300">Error: {res.error}</p>}
                </div>
              ))}
            </div>
          )}
        </div>
      </section>

      {/* Session Conflict Warning */}
      {sessionConflicts.length > 0 && (
        <section className="rounded-2xl border border-purple-400/50 bg-purple-950/20 p-5">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 mt-0.5 text-purple-400">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-lg font-semibold text-purple-100">
                Session Spawn Conflict
              </h3>
              <p className="mt-1 text-sm text-purple-200/80">
                You already spawned agents for overlapping paths in this browser session.
                This prevents accidentally spawning duplicate agents.
              </p>

              <div className="mt-4 space-y-2">
                {sessionConflicts.map((spawn, idx) => (
                  <div
                    key={`${spawn.scenario}-${spawn.spawnedAt}-${idx}`}
                    className="rounded-xl border border-purple-400/30 bg-purple-950/30 p-3"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-purple-100">
                        {spawn.scenario}
                      </span>
                      <span className="text-xs text-purple-300/70">
                        {new Date(spawn.spawnedAt).toLocaleTimeString()}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {spawn.scope.length > 0 ? (
                        spawn.scope.slice(0, 4).map((path) => (
                          <span
                            key={path}
                            className="rounded-full border border-purple-400/30 bg-purple-400/10 px-2 py-0.5 text-[10px] text-purple-100 font-mono"
                          >
                            {path}
                          </span>
                        ))
                      ) : (
                        <span className="text-xs text-purple-300/70 italic">
                          Entire scenario
                        </span>
                      )}
                      {spawn.scope.length > 4 && (
                        <span className="text-xs text-purple-400/60">
                          +{spawn.scope.length - 4} more
                        </span>
                      )}
                    </div>
                    <div className="mt-2 text-xs text-purple-300/60">
                      Agents: {spawn.agentIds.join(", ")}
                    </div>
                  </div>
                ))}
              </div>

              <div className="mt-4 flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => {
                    // Mark conflicting spawns as completed to allow re-spawning
                    setSessionSpawns((prev) => {
                      const updated = prev.map((spawn) => {
                        const isConflicting = sessionConflicts.some(
                          (c) => c.spawnedAt === spawn.spawnedAt && c.scenario === spawn.scenario
                        );
                        if (isConflicting) {
                          return { ...spawn, status: "completed" as const };
                        }
                        return spawn;
                      });
                      saveSessionSpawns(updated);
                      return updated;
                    });
                    setSessionConflicts([]);
                  }}
                  className="rounded-lg border border-purple-400/40 bg-purple-400/10 px-3 py-1.5 text-sm text-purple-100 hover:bg-purple-400/20 transition-colors"
                >
                  Clear conflicts & allow re-spawn
                </button>
                <span className="text-xs text-purple-300/60">
                  Use if agents have completed or you want to spawn additional agents
                </span>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Pre-Spawn Conflict Preview */}
      <ConflictPreview
        conflicts={scopeConflicts}
        onStopAgent={handleConflictAgentStopped}
        onRefresh={checkForConflicts}
      />

      {/* Active Agents Panel */}
      <ActiveAgentsPanel
        scenario={focusScenario}
        scope={targetPaths}
        onConflictDetected={handleConflictDetected}
        onAgentStatusChange={(agentId, status) => {
          // Update session spawn tracking when agents complete
          const terminalStatus: SessionSpawnRecord["status"] = status === "completed" ? "completed" : "failed";
          setSessionSpawns((prev) => {
            const updated: SessionSpawnRecord[] = prev.map((spawn) => {
              if (spawn.agentIds.includes(agentId) && spawn.status === "active") {
                // Remove this agent from the list
                const remainingAgents = spawn.agentIds.filter((id) => id !== agentId);
                // If no agents remain, mark the spawn as completed/failed
                if (remainingAgents.length === 0) {
                  return { ...spawn, status: terminalStatus, agentIds: remainingAgents };
                }
                // Otherwise just update the agent list
                return { ...spawn, agentIds: remainingAgents };
              }
              return spawn;
            });
            saveSessionSpawns(updated);
            return updated;
          });
        }}
      />

      {/* Action Buttons */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Actions</p>
        <h3 className="mt-2 text-lg font-semibold">Use this prompt</h3>
        <p className="mt-2 mb-4 text-sm text-slate-300">
          Copy the prompt to use with Claude, ChatGPT, or another AI assistant.
        </p>
        <ActionButtons
          prompt={currentPromptItem?.prompt ?? ""}
          allPrompts={promptItems.map((item) => item.prompt)}
          disabled={isDisabled}
          spawnDisabled={spawnDisabled || scopeConflicts.length > 0 || sessionConflicts.length > 0}
          spawnBusy={spawnBusy}
          onSpawnAll={handleSpawnAll}
        />
      </section>

      <ScenarioTargetDialog
        open={isDialogOpen}
        onClose={() => setIsDialogOpen(false)}
        scenarios={scenarioDirectoryEntries}
        initialScenario={focusScenario}
        initialTargets={targetPaths}
        onSave={handleScopeSave}
      />
    </div>
  );
}

export { PhaseSelector, PromptEditor, ActionButtons, PresetSelector };
