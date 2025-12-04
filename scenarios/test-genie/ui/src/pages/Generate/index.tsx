import { useEffect, useMemo, useState } from "react";
import { PhaseSelector } from "./PhaseSelector";
import { PromptEditor } from "./PromptEditor";
import { ActionButtons } from "./ActionButtons";
import { PresetSelector } from "./PresetSelector";
import { ScenarioTargetDialog } from "./ScenarioTargetDialog";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import { PHASES_FOR_GENERATION, PHASE_LABELS, REPO_ROOT } from "../../lib/constants";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { AgentModel, SpawnAgentsResult, fetchAgentModels, spawnAgents } from "../../lib/api";

const MAX_PROMPTS = 12;
const RECENT_MODEL_STORAGE_KEY = "test-genie-recent-agent-models";

type PromptCombination = {
  id: string;
  index: number;
  total: number;
  phases: string[];
  targets: string[];
  prompt: string;
  defaultPrompt: string;
  label: string;
};

const comboId = (targets: string[], phases: string[]) =>
  `t:${targets.length > 0 ? targets.join("|") : "all"}|p:${phases.join("|")}`;

function buildPrompt(
  scenarioName: string,
  selectedPhases: string[],
  preset: string | null,
  targetPaths: string[]
): string {
  if (!scenarioName || selectedPhases.length === 0) {
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

  const scenarioPath = `${REPO_ROOT}/scenarios/${scenarioName}`;
  const scenarioDocsPath = `${scenarioPath}/docs`;
  const prdPath = `${scenarioPath}/PRD.md`;
  const requirementsPath = `${scenarioPath}/requirements`;
  const generalTestingDoc = `${REPO_ROOT}/scenarios/test-genie/docs/guides/test-generation.md`;
  const phaseDocs = selectedPhases
    .map((phase) => PHASES_FOR_GENERATION.find((p) => p.key === phase))
    .filter(Boolean)
    .map((phase) => `${REPO_ROOT}/scenarios/test-genie${phase!.docsPath}`);

  const phaseDocsList =
    phaseDocs.length > 0 ? phaseDocs.map((doc) => `- ${doc}`).join("\n") : "- (no phase docs selected)";

  const absoluteTargets = hasTargets
    ? targetPaths.map((target) => `${REPO_ROOT}/scenarios/${scenarioName}/${target}`)
    : [];

  return `Generate tests for the "${scenarioName}" scenario.

**Test phases:** ${phaseLabels}

${context ? `**Preset context:** ${context}\n\n` : ""}**Paths (absolute):**
- Scenario root: ${scenarioPath}
- PRD: ${prdPath}
- Requirements: ${requirementsPath}
- Scenario docs (if present): ${scenarioDocsPath}

**Docs to review first (absolute):**
- General test generation guide: ${generalTestingDoc}
- Phase docs for selected phases:
${phaseDocsList}

${hasTargets ? `**Targeted scope:**\n${absoluteTargets.map((p) => `- ${p}`).join("\n")}\n\n**Focus:** Add or enhance tests only for the components/files above. Avoid touching other paths, configs, or infrastructure.` : ""}
**MUST:**
- Align to PRD/requirements and existing coding/testing conventions in the scenario
- Make tests deterministic, isolated, and idempotent (no hidden state, fixed seeds, stable selectors)
- Add clear assertions and negative cases; keep names descriptive
- Use existing helpers/fixtures; prefer real interfaces for integration, limited mocking only where already used
- Mark requirement coverage if applicable (e.g., \`[REQ:ID]\`) and keep coverage neutral or improved
- Call out any assumptions and uncertain areas explicitly

**NEVER:**
- Touch files outside the scenario root
- Add dependencies or change runtime configs
- Delete or weaken existing tests without explicit rationale
- Introduce flaky behavior (timing sleeps, random data without seeds, network calls to external services)

**Generation steps:**
1) Read PRD, requirements, and existing tests to mirror patterns
2) For each selected phase, target meaningful coverage:
   - Unit: small, fast, local; no network/filesystem unless already mocked
   - Integration: exercise real component boundaries; minimal mocking; preserve setup/teardown hygiene
   - Playbooks (E2E): stable selectors, retries around navigation, capture screenshots/logs on failure hooks
   - Business: assertions tied to requirements/PRD acceptance criteria
3) Create or update tests under the scenario using existing structure and naming${hasTargets ? "; limit changes to the targeted paths above" : ""}
4) Keep changes idempotent and documented (comments only where intent is non-obvious)

**Validation (run after generation):**
- Execute via test-genie with the selected phases:
  /home/matthalloran8/Vrooli/scenarios/test-genie/cli/test-genie execute ${scenarioName} --phases ${selectedPhases.join(",")} --sync
- If preset implies broader coverage, also run:
  /home/matthalloran8/Vrooli/scenarios/test-genie/cli/test-genie execute ${scenarioName} --preset comprehensive --sync
- If any failures occur, include logs/output snippets and suggested fixes.

**Return (concise):**
- Files created/modified (with short rationale)
- Commands you ran and their results
- Any blockers, open questions, or assumptions that need confirmation

Please generate comprehensive ${phaseLabels.toLowerCase()} for this scenario while following the above constraints.`;
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
  const [allowedTools, setAllowedTools] = useState("edit,write");
  const [skipPermissions, setSkipPermissions] = useState(false);
  const [spawnBusy, setSpawnBusy] = useState(false);
  const [spawnStatus, setSpawnStatus] = useState<string | null>(null);
  const [spawnResults, setSpawnResults] = useState<SpawnAgentsResult[] | null>(null);

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
    return cappedCombos.map((combo, index) => {
      const id = comboId(combo.targets, combo.phases);
      const defaultPrompt = buildPrompt(focusScenario, combo.phases, selectedPreset, combo.targets);
      const prompt = customPrompts[id] ?? defaultPrompt;
      const targetLabel =
        combo.targets.length === 0 ? "All paths" : combo.targets.join(", ");
      const phaseLabel = combo.phases.map((p) => PHASE_LABELS[p] ?? p).join(", ");
      return {
        id,
        index,
        total: cappedCombos.length,
        phases: combo.phases,
        targets: combo.targets,
        prompt,
        defaultPrompt,
        label: `Prompt ${index + 1} • ${targetLabel} • ${phaseLabel}`
      };
    });
  }, [cappedCombos, customPrompts, focusScenario, selectedPreset]);

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

  const handlePromptChange = (id: string, value: string, defaultValue: string) => {
    setCustomPrompts((prev) => {
      const next = { ...prev };
      if (value === defaultValue) {
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
    setSpawnStatus("Spawning agents via OpenCode/OpenRouter...");
    setSpawnResults(null);
    try {
      const allowed = allowedTools
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);
      const res = await spawnAgents({
        prompts: promptItems.map((p) => p.prompt),
        model: agentModel,
        concurrency: Math.max(1, Math.min(safeConcurrency, promptCount)),
        maxTurns: safeMaxTurns > 0 ? safeMaxTurns : undefined,
        timeoutSeconds: safeTimeout > 0 ? safeTimeout : undefined,
        allowedTools: allowed.length ? allowed : undefined,
        skipPermissions
      });
      setSpawnResults(res.items ?? []);
      saveRecentModel(agentModel);
      const cappedNote = res.capped ? " (first batch capped; narrow scope to spawn all)" : "";
      setSpawnStatus(`Spawned ${res.items.length} agent${res.items.length === 1 ? "" : "s"}${cappedNote}.`);
    } catch (err) {
      setSpawnStatus(err instanceof Error ? err.message : "Failed to spawn agents");
    } finally {
      setSpawnBusy(false);
    }
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
            prompt={currentPromptItem?.prompt ?? ""}
            onPromptChange={(value) =>
              currentPromptItem && handlePromptChange(currentPromptItem.id, value, currentPromptItem.defaultPrompt)
            }
            defaultPrompt={currentPromptItem?.defaultPrompt ?? ""}
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
              <input
                type="number"
                min={1}
                max={10}
                value={spawnConcurrency}
                onChange={(e) => setSpawnConcurrency(Number(e.target.value))}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
              />
              <p className="text-[11px] text-slate-400 mt-1">Max parallel agent runs (cap 10).</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Max turns</label>
              <input
                type="number"
                min={0}
                value={maxTurns}
                onChange={(e) => setMaxTurns(Number(e.target.value))}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
              />
              <p className="text-[11px] text-slate-400 mt-1">0 = unlimited (not recommended).</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Timeout (sec)</label>
              <input
                type="number"
                min={0}
                value={timeoutSeconds}
                onChange={(e) => setTimeoutSeconds(Number(e.target.value))}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
              />
              <p className="text-[11px] text-slate-400 mt-1">Abort if a request exceeds this duration.</p>
            </div>
            <div>
              <label className="text-xs uppercase tracking-[0.2em] text-slate-500">Allowed tools</label>
              <input
                type="text"
                value={allowedTools}
                onChange={(e) => setAllowedTools(e.target.value)}
                className="mt-1 w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                placeholder="edit,write"
              />
              <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-400">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    className="h-4 w-4"
                    checked={skipPermissions}
                    onChange={(e) => setSkipPermissions(e.target.checked)}
                  />
                  Auto-approve permissions (unsafe)
                </label>
              </div>
            </div>
          </div>
        </div>

        <div className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4 text-sm text-slate-200">
          <p className="font-semibold text-white">Dispatch summary</p>
          <p className="mt-1 text-slate-300">
            {promptCount} prompt{promptCount === 1 ? "" : "s"} • model {agentModel || "not selected"} • concurrency{" "}
            {safeDisplayConcurrency}
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
          spawnDisabled={spawnDisabled}
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
