import { useState, useMemo, useId } from "react";
import { PhaseSelector } from "./PhaseSelector";
import { PromptEditor } from "./PromptEditor";
import { ActionButtons } from "./ActionButtons";
import { PresetSelector } from "./PresetSelector";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import { PHASES_FOR_GENERATION, PHASE_LABELS } from "../../lib/constants";

function buildPrompt(
  scenarioName: string,
  selectedPhases: string[],
  preset: string | null
): string {
  if (!scenarioName || selectedPhases.length === 0) {
    return "";
  }

  const phaseLabels = selectedPhases
    .map((p) => PHASE_LABELS[p] ?? p)
    .join(", ");

  let context = "";
  if (preset === "bootstrap") {
    context = `This is a new scenario that needs initial test coverage. Focus on establishing a solid foundation of tests.`;
  } else if (preset === "coverage") {
    context = `The scenario already has some tests. Focus on adding coverage for specific features or edge cases.`;
  } else if (preset === "fix-failing") {
    context = `Some tests are failing. Focus on understanding the failures and generating fixes or improved test cases.`;
  }

  return `Generate tests for the "${scenarioName}" scenario.

**Test Phases:** ${phaseLabels}

${context ? `**Context:** ${context}\n\n` : ""}**Requirements:**
1. Follow the existing code patterns and conventions in the scenario
2. Write clear, maintainable test code with descriptive names
3. Cover both happy paths and edge cases
4. Include appropriate assertions and error handling
5. Add comments explaining the test intent where helpful

**Instructions:**
1. First, explore the scenario structure at \`scenarios/${scenarioName}/\`
2. Identify the main components, functions, or modules to test
3. Generate test files following the scenario's test conventions
4. Run the tests to verify they pass

Please generate comprehensive ${phaseLabels.toLowerCase()} for this scenario.`;
}

export function GeneratePage() {
  const datalistId = useId();
  const { scenarioDirectoryEntries } = useScenarios();
  const { focusScenario, setFocusScenario } = useUIStore();

  const [selectedPhases, setSelectedPhases] = useState<string[]>(["unit", "integration"]);
  const [selectedPreset, setSelectedPreset] = useState<string | null>(null);
  const [customPrompt, setCustomPrompt] = useState<string | null>(null);

  const scenarioOptions = useMemo(
    () => scenarioDirectoryEntries.map((s) => s.scenarioName),
    [scenarioDirectoryEntries]
  );

  const defaultPrompt = useMemo(
    () => buildPrompt(focusScenario, selectedPhases, selectedPreset),
    [focusScenario, selectedPhases, selectedPreset]
  );

  const currentPrompt = customPrompt ?? defaultPrompt;

  const handleTogglePhase = (phase: string) => {
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

      {/* Scenario Selection */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Target</p>
        <h3 className="mt-2 text-lg font-semibold">Select scenario</h3>
        <p className="mt-2 text-sm text-slate-300">
          Choose the scenario you want to generate tests for.
        </p>

        <datalist id={datalistId}>
          {scenarioOptions.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>

        <input
          className="mt-4 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
          placeholder="Enter scenario name..."
          value={focusScenario}
          onChange={(e) => setFocusScenario(e.target.value)}
          list={datalistId}
        />

        {scenarioOptions.length > 0 && !focusScenario && (
          <div className="mt-3 flex flex-wrap gap-2">
            <span className="text-xs text-slate-400">Quick select:</span>
            {scenarioOptions.slice(0, 5).map((name) => (
              <button
                key={name}
                type="button"
                onClick={() => setFocusScenario(name)}
                className="rounded-full border border-white/20 px-3 py-1 text-xs text-slate-300 hover:border-white/50 transition"
              >
                {name}
              </button>
            ))}
          </div>
        )}
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
    </div>
  );
}

export { PhaseSelector, PromptEditor, ActionButtons, PresetSelector };
