import { useState, useMemo } from "react";
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
  const [customPrompt, setCustomPrompt] = useState<string | null>(null);
  const [targetPaths, setTargetPaths] = useState<string[]>([]);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const defaultPrompt = useMemo(
    () => buildPrompt(focusScenario, selectedPhases, selectedPreset, targetPaths),
    [focusScenario, selectedPhases, selectedPreset, targetPaths]
  );

  const currentPrompt = customPrompt ?? defaultPrompt;
  const hasTargets = targetPaths.length > 0;

  const handleTogglePhase = (phase: string) => {
    if (hasTargets) return; // When targeting paths, phases are locked to Unit
    setSelectedPhases((prev) =>
      prev.includes(phase)
        ? prev.filter((p) => p !== phase)
        : [...prev, phase]
    );
    setCustomPrompt(null); // Reset custom prompt when phases change
  };

  const handlePresetSelect = (preset: string) => {
    setSelectedPreset((prev) => (prev === preset ? null : preset));
    setCustomPrompt(null);
  };

  const handlePromptChange = (prompt: string) => {
    setCustomPrompt(prompt);
  };

  const handleScopeSave = (scenario: string, paths: string[]) => {
    setFocusScenario(scenario);
    setTargetPaths(paths);
    setCustomPrompt(null);
    if (paths.length > 0) {
      setSelectedPhases(["unit"]);
    } else {
      setSelectedPhases(["unit", "integration"]);
    }
  };

  const isDisabled = !focusScenario || selectedPhases.length === 0;

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
      </section>

      {/* Prompt Preview/Editor */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <PromptEditor
          prompt={currentPrompt}
          onPromptChange={handlePromptChange}
          defaultPrompt={defaultPrompt}
        />
      </section>

      {/* Action Buttons */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Actions</p>
        <h3 className="mt-2 text-lg font-semibold">Use this prompt</h3>
        <p className="mt-2 mb-4 text-sm text-slate-300">
          Copy the prompt to use with Claude, ChatGPT, or another AI assistant.
        </p>
        <ActionButtons prompt={currentPrompt} disabled={isDisabled} />
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
