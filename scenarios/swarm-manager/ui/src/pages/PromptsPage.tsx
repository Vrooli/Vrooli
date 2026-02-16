import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { selectors } from "../consts/selectors";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { promptService } from "../services";
import type { BacklogKind, IdeaAgentMode, PromptSkillVersion } from "../types";

const KINDS: BacklogKind[] = ["idea", "research", "fix", "execute"];
const MODES: Array<IdeaAgentMode | "research"> = ["clarify", "suggest", "enhance", "research"];
const OPERATIONS: Array<"" | "generator" | "improver"> = ["", "generator", "improver"];

const splitLines = (value: string) => value.replace(/\r\n/g, "\n").split("\n");

const buildSimpleDiff = (next: string, previous: string): string[] => {
  const nextLines = splitLines(next);
  const previousLines = splitLines(previous);
  const max = Math.max(nextLines.length, previousLines.length);
  const output: string[] = [];
  for (let i = 0; i < max; i++) {
    const before = previousLines[i] ?? "";
    const after = nextLines[i] ?? "";
    if (before === after) {
      output.push(`  ${after}`);
      continue;
    }
    if (before !== "") output.push(`- ${before}`);
    if (after !== "") output.push(`+ ${after}`);
  }
  return output;
};

export function PromptsPage() {
  const queryClient = useQueryClient();
  const [selectedSkillId, setSelectedSkillId] = useState<string>("");
  const [content, setContent] = useState("");
  const [simulationKind, setSimulationKind] = useState<BacklogKind>("idea");
  const [simulationMode, setSimulationMode] = useState<IdeaAgentMode | "research">("clarify");
  const [simulationOperation, setSimulationOperation] = useState<"" | "generator" | "improver">("");
  const [comparisonVersion, setComparisonVersion] = useState<PromptSkillVersion | null>(null);

  const bindingsQuery = useQuery({
    queryKey: ["prompts", "bindings"],
    queryFn: () => promptService.listBindings(),
    ...defaultQueryOptions,
  });

  const skillsQuery = useQuery({
    queryKey: ["prompts", "skills"],
    queryFn: () => promptService.listSkills(),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    const firstSkill = skillsQuery.data?.[0];
    if (!selectedSkillId && firstSkill) {
      setSelectedSkillId(firstSkill.id);
    }
  }, [selectedSkillId, skillsQuery.data]);

  const skillQuery = useQuery({
    queryKey: ["prompts", "skill", selectedSkillId],
    queryFn: () => promptService.getSkill(selectedSkillId),
    enabled: selectedSkillId.length > 0,
    ...defaultQueryOptions,
  });

  const versionsQuery = useQuery({
    queryKey: ["prompts", "versions", selectedSkillId],
    queryFn: () => promptService.getSkillVersions(selectedSkillId),
    enabled: selectedSkillId.length > 0,
    ...defaultQueryOptions,
  });

  useEffect(() => {
    if (skillQuery.data?.current_content !== undefined) {
      setContent(skillQuery.data.current_content);
    }
  }, [skillQuery.data?.current_content]);

  const updateMutation = useMutation({
    mutationFn: ({
      draft,
      nextContent,
    }: {
      draft: boolean;
      nextContent: string;
    }) =>
      promptService.updateSkill(selectedSkillId, {
        content: nextContent,
        draft,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["prompts", "skills"] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "skill", selectedSkillId] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "versions", selectedSkillId] });
    },
  });

  const revertMutation = useMutation({
    mutationFn: (version: number) => promptService.revertSkillVersion(selectedSkillId, version),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["prompts", "skills"] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "skill", selectedSkillId] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "versions", selectedSkillId] });
      setComparisonVersion(null);
    },
  });

  const simulateMutation = useMutation({
    mutationFn: () =>
      promptService.simulate({
        kind: simulationKind,
        mode: simulationMode,
        operation: simulationOperation || undefined,
        item_name: "sample-item",
        item_title: "Sample Item",
        item_description: "Sample description for simulation preview.",
        item_status: "backlog",
        item_priority: "3",
        item_tags: "sample, prompt-center",
        item_folder: "scenarios/swarm-manager/ideas/sample-item",
      }),
  });

  const selectedSkill = skillQuery.data;

  const diffLines = useMemo(() => {
    if (!comparisonVersion) return [];
    return buildSimpleDiff(content, comparisonVersion.content);
  }, [comparisonVersion, content]);

  if (bindingsQuery.isLoading || skillsQuery.isLoading) {
    return <PageLoadingState variant="settings" message="Loading prompt center..." />;
  }

  if (bindingsQuery.error || skillsQuery.error) {
    const err = bindingsQuery.error ?? skillsQuery.error;
    return (
      <ErrorState
        title="Unable to load prompts"
        message={err instanceof Error ? err.message : "Prompt center failed to load."}
        onRetry={() => {
          void bindingsQuery.refetch();
          void skillsQuery.refetch();
        }}
      />
    );
  }

  return (
    <div className="space-y-6" data-testid={selectors.prompts.page}>
      <Card className="space-y-3 p-4" data-testid={selectors.prompts.bindingMap}>
        <h2 className="text-lg font-semibold text-slate-100">Prompt Map</h2>
        <p className="text-sm text-slate-400">
          This map shows when each prompt runs and what output it is expected to produce.
        </p>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px] text-left text-sm">
            <thead className="text-slate-400">
              <tr>
                <th className="px-2 py-2">Area</th>
                <th className="px-2 py-2">Trigger</th>
                <th className="px-2 py-2">Kind / Mode</th>
                <th className="px-2 py-2">Prompt</th>
                <th className="px-2 py-2">Purpose</th>
              </tr>
            </thead>
            <tbody>
              {(bindingsQuery.data ?? []).map((binding) => (
                <tr key={`${binding.skill_id}-${binding.trigger}`} className="border-t border-slate-700/60 text-slate-200">
                  <td className="px-2 py-2 uppercase text-xs text-slate-400">{binding.area}</td>
                  <td className="px-2 py-2">{binding.trigger}</td>
                  <td className="px-2 py-2 text-slate-300">
                    {binding.kind ?? "-"} / {binding.mode ?? binding.operation ?? "-"}
                  </td>
                  <td className="px-2 py-2">
                    <button
                      className="font-mono text-cyan-300 hover:text-cyan-200"
                      onClick={() => setSelectedSkillId(binding.skill_id)}
                    >
                      {binding.skill_id}
                    </button>
                  </td>
                  <td className="px-2 py-2 text-slate-300">{binding.purpose}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <div className="grid gap-4 lg:grid-cols-[320px_1fr]">
        <Card className="space-y-3 p-4" data-testid={selectors.prompts.skillsList}>
          <h3 className="text-base font-semibold text-slate-100">Swarm Prompt Skills</h3>
          <div className="space-y-2">
            {(skillsQuery.data ?? []).map((skill) => (
              <button
                key={skill.id}
                className={`w-full rounded-md border px-3 py-2 text-left transition ${
                  selectedSkillId === skill.id
                    ? "border-cyan-500/60 bg-cyan-500/10"
                    : "border-slate-700/60 bg-slate-900/30 hover:border-slate-500/60"
                }`}
                onClick={() => setSelectedSkillId(skill.id)}
              >
                <p className="font-mono text-xs text-cyan-300">{skill.id}</p>
                <p className="mt-1 text-sm text-slate-100">{skill.name}</p>
                <p className="mt-1 text-xs text-slate-400">{skill.impact_summary}</p>
              </button>
            ))}
          </div>
        </Card>

        <Card className="space-y-4 p-4" data-testid={selectors.prompts.editor}>
          {skillQuery.isLoading ? (
            <InlineLoadingIndicator message="Loading prompt skill..." />
          ) : selectedSkill ? (
            <>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h3 className="font-mono text-sm text-cyan-300">{selectedSkill.id}</h3>
                  <p className="text-xs text-slate-400">
                    Updated {selectedSkill.updated_at ? formatRelativeTime(selectedSkill.updated_at) : "unknown"} •{" "}
                    {selectedSkill.impact_summary}
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    data-testid={selectors.prompts.saveDraft}
                    disabled={updateMutation.isPending}
                    onClick={() => updateMutation.mutate({ draft: true, nextContent: content })}
                  >
                    Save Draft
                  </Button>
                  <Button
                    size="sm"
                    data-testid={selectors.prompts.publish}
                    disabled={updateMutation.isPending}
                    onClick={() => updateMutation.mutate({ draft: false, nextContent: content })}
                  >
                    Publish
                  </Button>
                </div>
              </div>

              {selectedSkill.required_missing && selectedSkill.required_missing.length > 0 ? (
                <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                  Missing required variables: {selectedSkill.required_missing.join(", ")}
                </div>
              ) : null}

              <textarea
                value={content}
                onChange={(event) => setContent(event.target.value)}
                className="min-h-[260px] w-full rounded-md border border-slate-700/70 bg-slate-950 px-3 py-2 font-mono text-xs text-slate-100 focus:border-cyan-500 focus:outline-none"
                data-testid={selectors.prompts.contentInput}
              />

              <div className="grid gap-4 xl:grid-cols-2">
                <div className="space-y-2" data-testid={selectors.prompts.versions}>
                  <h4 className="text-sm font-semibold text-slate-100">Version History</h4>
                  <div className="max-h-56 space-y-2 overflow-auto pr-1">
                    {(versionsQuery.data?.versions ?? []).map((version) => (
                      <div key={version.version} className="rounded-md border border-slate-700/60 bg-slate-900/40 p-2">
                        <p className="text-xs text-slate-200">
                          v{version.version} • {formatRelativeTime(version.updatedAt)}
                        </p>
                        <div className="mt-2 flex gap-2">
                          <Button size="sm" variant="outline" onClick={() => setComparisonVersion(version)}>
                            Compare
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={revertMutation.isPending}
                            onClick={() => revertMutation.mutate(version.version)}
                          >
                            Rollback
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="space-y-2" data-testid={selectors.prompts.preview}>
                  <h4 className="text-sm font-semibold text-slate-100">Simulation Preview</h4>
                  <div className="grid grid-cols-3 gap-2">
                    <Select value={simulationKind} onChange={(event) => setSimulationKind(event.target.value as BacklogKind)}>
                      {KINDS.map((kind) => (
                        <option key={kind} value={kind}>{kind}</option>
                      ))}
                    </Select>
                    <Select value={simulationMode} onChange={(event) => setSimulationMode(event.target.value as IdeaAgentMode | "research")}>
                      {MODES.map((mode) => (
                        <option key={mode} value={mode}>{mode}</option>
                      ))}
                    </Select>
                    <Select value={simulationOperation} onChange={(event) => setSimulationOperation(event.target.value as "" | "generator" | "improver")}>
                      {OPERATIONS.map((operation) => (
                        <option key={operation || "none"} value={operation}>
                          {operation || "research"}
                        </option>
                      ))}
                    </Select>
                  </div>
                  <Button size="sm" onClick={() => simulateMutation.mutate()}>
                    Simulate Trigger
                  </Button>
                  <div className="max-h-56 overflow-auto rounded-md border border-slate-700/60 bg-slate-950 p-2">
                    {simulateMutation.isPending ? (
                      <InlineLoadingIndicator message="Resolving prompt..." />
                    ) : simulateMutation.data ? (
                      <pre className="whitespace-pre-wrap break-words font-mono text-xs text-slate-100">
                        {simulateMutation.data.prompt}
                      </pre>
                    ) : (
                      <p className="text-xs text-slate-400">Run a simulation to preview the exact prompt text.</p>
                    )}
                  </div>
                </div>
              </div>

              {comparisonVersion ? (
                <div className="space-y-2">
                  <h4 className="text-sm font-semibold text-slate-100">
                    Diff vs v{comparisonVersion.version}
                  </h4>
                  <div className="max-h-56 overflow-auto rounded-md border border-slate-700/60 bg-slate-950 p-2">
                    <pre className="whitespace-pre-wrap break-words font-mono text-xs text-slate-200">
                      {diffLines.join("\n")}
                    </pre>
                  </div>
                </div>
              ) : null}
            </>
          ) : (
            <p className="text-sm text-slate-400">Select a prompt skill to inspect and edit.</p>
          )}
        </Card>
      </div>
    </div>
  );
}
