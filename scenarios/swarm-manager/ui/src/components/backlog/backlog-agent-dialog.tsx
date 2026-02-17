// DOC: docs/guides/idea-agent-workflow.md
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
import { useEffect, useMemo, useState } from "react";
import { X, Sparkles, ChevronDown, ChevronRight, Paperclip, CheckCircle2, Circle } from "lucide-react";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { FileTree } from "../ui/file-tree";
import { selectors } from "../../consts/selectors";
import { IDEA_AGENT_FILE_PATHS } from "../../lib";
import {
  BACKLOG_KIND_LABELS,
  BACKLOG_RESEARCH_TARGET_LABELS,
  BACKLOG_RESEARCH_TARGETS,
  type BacklogKind,
  type BacklogFile,
  type BacklogResearchTarget,
  type IdeaAgentMode,
  type ArchiveTarget,
  type ArchiveRequirementGroup,
  type ArchiveTargetsResponse,
} from "../../types";

interface BacklogAgentDialogProps {
  isOpen: boolean;
  isSubmitting?: boolean;
  backlogKind: BacklogKind;
  backlogTitle: string;
  researchTarget?: BacklogResearchTarget;
  errorMessage?: string | null;
  files?: BacklogFile[];
  archiveTargets?: ArchiveTargetsResponse;
  onClose: () => void;
  onSubmit: (payload: {
    mode?: IdeaAgentMode;
    prompt: string;
    targetKind?: BacklogResearchTarget;
    contextPaths?: string[];
    contextTargetIds?: string[];
    contextRequirementIds?: string[];
  }) => void;
}

const MODE_OPTIONS: Array<{
  value: IdeaAgentMode;
  title: string;
  description: string;
  output: string;
}> = [
  {
    value: "clarify",
    title: "Clarify",
    description:
      "Gather the most relevant questions needed to clarify scope, constraints, and implementation details.",
    output: `Writes questions to ${IDEA_AGENT_FILE_PATHS.clarify}`,
  },
  {
    value: "suggest",
    title: "Suggest",
    description:
      "Generate improvements and alternative approaches for the idea, ready for review and selection.",
    output: `Writes suggestions to ${IDEA_AGENT_FILE_PATHS.suggest}`,
  },
  {
    value: "enhance",
    title: "Enhance",
    description:
      "Produce a refined plan using answered clarifications and accepted suggestions.",
    output: `Writes enhancements to ${IDEA_AGENT_FILE_PATHS.enhance}`,
  },
];

const RESEARCH_TARGET_OPTIONS: Array<{ value: BacklogResearchTarget; label: string }> =
  BACKLOG_RESEARCH_TARGETS.map((value) => ({
    value,
    label: BACKLOG_RESEARCH_TARGET_LABELS[value],
  }));

function RequirementCheckboxGroup({
  groups,
  selectedIds,
  onToggle,
  depth = 0,
}: {
  groups: ArchiveRequirementGroup[];
  selectedIds: Set<string>;
  onToggle: (id: string) => void;
  depth?: number;
}) {
  return (
    <div className={depth > 0 ? "ml-3 border-l border-slate-700/50 pl-2" : ""}>
      {groups.map((group) => (
        <RequirementCheckboxNode key={group.id} group={group} selectedIds={selectedIds} onToggle={onToggle} depth={depth} />
      ))}
    </div>
  );
}

function RequirementCheckboxNode({
  group,
  selectedIds,
  onToggle,
  depth,
}: {
  group: ArchiveRequirementGroup;
  selectedIds: Set<string>;
  onToggle: (id: string) => void;
  depth: number;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasContent = group.requirements.length > 0 || group.children.length > 0;

  if (!hasContent) return null;

  return (
    <div>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-1.5 rounded-md px-1 py-1 text-left text-xs font-medium text-slate-300 hover:bg-slate-800/50"
      >
        {expanded ? <ChevronDown className="h-3 w-3 text-slate-500" /> : <ChevronRight className="h-3 w-3 text-slate-500" />}
        <span>{group.name}</span>
        <span className="text-slate-500">({group.requirements.length})</span>
      </button>
      {expanded && (
        <div className="space-y-0.5">
          {group.requirements.map((req) => (
            <label
              key={req.id}
              className="flex items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-slate-800/50 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedIds.has(req.id)}
                onChange={() => onToggle(req.id)}
                className="mt-0.5 h-3.5 w-3.5 accent-cyan-500"
              />
              <div className="min-w-0 flex-1">
                <span className="font-mono text-slate-500">{req.id}</span>
                <span className="ml-1 text-slate-300">{req.title}</span>
              </div>
            </label>
          ))}
          {group.children.length > 0 && (
            <RequirementCheckboxGroup groups={group.children} selectedIds={selectedIds} onToggle={onToggle} depth={depth + 1} />
          )}
        </div>
      )}
    </div>
  );
}

export function BacklogAgentDialog({
  isOpen,
  isSubmitting = false,
  backlogKind,
  backlogTitle,
  researchTarget,
  errorMessage = null,
  files,
  archiveTargets,
  onClose,
  onSubmit,
}: BacklogAgentDialogProps) {
  const [prompt, setPrompt] = useState("");
  const [mode, setMode] = useState<IdeaAgentMode>("clarify");
  const [targetKind, setTargetKind] = useState<BacklogResearchTarget>(researchTarget ?? "idea");
  const [selectedFilePaths, setSelectedFilePaths] = useState<Set<string>>(new Set());
  const [selectedTargetIds, setSelectedTargetIds] = useState<Set<string>>(new Set());
  const [selectedRequirementIds, setSelectedRequirementIds] = useState<Set<string>>(new Set());
  const [showAttachContext, setShowAttachContext] = useState(false);
  const [attachTab, setAttachTab] = useState<"files" | "targets">("files");

  const isIdea = backlogKind === "idea";
  const isResearch = backlogKind === "research";

  const totalAttached = selectedFilePaths.size + selectedTargetIds.size + selectedRequirementIds.size;

  const hasArchive = archiveTargets?.has_archive ?? false;
  const archiveTargetList = archiveTargets?.targets ?? [];
  const archiveRequirements = archiveTargets?.requirements ?? [];
  const hasContextOptions = (files && files.length > 0) || hasArchive;

  useEffect(() => {
    if (isOpen) {
      setPrompt("");
      setMode("clarify");
      setTargetKind(researchTarget ?? "idea");
      setSelectedFilePaths(new Set());
      setSelectedTargetIds(new Set());
      setSelectedRequirementIds(new Set());
      setShowAttachContext(false);
      setAttachTab("files");
    }
  }, [isOpen, researchTarget]);

  const activeMode = useMemo(() => MODE_OPTIONS.find((option) => option.value === mode), [mode]);

  if (!isOpen) return null;

  const title = isIdea ? "Run Idea Agent" : "Run Research Agent";
  const description = isIdea
    ? "This agent will update files inside the idea folder so you can review, edit, and build on the output."
    : "This agent will update files inside the backlog folder with a research summary and supporting notes.";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid={selectors.backlogForm.agentDialog}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <div className="relative z-10 w-full max-w-2xl rounded-xl border border-white/10 bg-slate-900 p-6 shadow-2xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-4 top-4 rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex items-center gap-3">
          <div className="rounded-full bg-cyan-500/20 p-2">
            <Sparkles className="h-5 w-5 text-cyan-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-slate-100">{title}</h2>
            <p className="text-sm text-slate-400">{BACKLOG_KIND_LABELS[backlogKind]} • {backlogTitle}</p>
          </div>
        </div>

        <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/40 p-3 text-sm text-slate-300">
          {description}
        </div>

        {isIdea && (
          <fieldset className="mt-4 space-y-3" data-testid={selectors.backlogForm.agentMode}>
            <legend className="text-sm font-medium text-slate-300">Agent type</legend>
            <div className="grid gap-3 md:grid-cols-3">
              {MODE_OPTIONS.map((option) => {
                const isSelected = option.value === mode;
                return (
                  <label
                    key={option.value}
                    className={`flex h-full cursor-pointer flex-col gap-2 rounded-lg border p-3 text-left transition ${
                      isSelected
                        ? "border-cyan-500/60 bg-cyan-500/10"
                        : "border-white/10 bg-slate-800/40 hover:border-white/20"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-semibold text-slate-100">{option.title}</span>
                      <input
                        type="radio"
                        name="idea-agent-mode"
                        value={option.value}
                        checked={isSelected}
                        onChange={() => setMode(option.value)}
                        className="h-4 w-4 accent-cyan-500"
                      />
                    </div>
                    <p className="text-xs text-slate-400">{option.description}</p>
                    <p className="text-xs text-slate-500">{option.output}</p>
                  </label>
                );
              })}
            </div>
          </fieldset>
        )}

        {isResearch && (
          <div className="mt-4">
            <label htmlFor="backlog-agent-target" className="text-sm font-medium text-slate-300">
              Research target
            </label>
            <div className="mt-2">
              <Select
                id="backlog-agent-target"
                value={targetKind}
                onChange={(e) => setTargetKind(e.target.value as BacklogResearchTarget)}
              >
                {RESEARCH_TARGET_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </div>
            <p className="mt-1 text-xs text-slate-500">
              Select what kind of backlog item this research should convert into later.
            </p>
          </div>
        )}

        <div className="mt-4 space-y-3">
          <label htmlFor="backlog-agent-context" className="text-sm font-medium text-slate-300">
            Additional context (optional)
          </label>
          <textarea
            id="backlog-agent-context"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Add any constraints, focus areas, or implementation notes."
            className="w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={4}
            data-testid={selectors.backlogForm.agentContext}
            disabled={isSubmitting}
          />
          {isIdea && activeMode && (
            <div className="text-xs text-slate-400">
              Next output: <span className="text-slate-200">{activeMode.output}</span>
            </div>
          )}
          {errorMessage && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {errorMessage}
            </div>
          )}
        </div>

        {hasContextOptions && (
          <div className="mt-4">
            <button
              type="button"
              onClick={() => setShowAttachContext(!showAttachContext)}
              className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-300 hover:border-white/20"
            >
              <Paperclip className="h-4 w-4 text-slate-400" />
              <span className="flex-1">Attach Context</span>
              {totalAttached > 0 && (
                <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs font-medium text-cyan-400">
                  {totalAttached} item{totalAttached !== 1 ? "s" : ""} attached
                </span>
              )}
              {showAttachContext ? (
                <ChevronDown className="h-4 w-4 text-slate-500" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-500" />
              )}
            </button>

            {showAttachContext && (
              <div className="mt-2 rounded-lg border border-white/10 bg-slate-800/30">
                <div className="flex border-b border-white/10">
                  {files && files.length > 0 && (
                    <button
                      type="button"
                      onClick={() => setAttachTab("files")}
                      className={`flex-1 px-3 py-2 text-xs font-medium transition ${
                        attachTab === "files"
                          ? "border-b-2 border-cyan-500 text-cyan-400"
                          : "text-slate-400 hover:text-slate-200"
                      }`}
                    >
                      Files
                    </button>
                  )}
                  {hasArchive && (
                    <button
                      type="button"
                      onClick={() => setAttachTab("targets")}
                      className={`flex-1 px-3 py-2 text-xs font-medium transition ${
                        attachTab === "targets"
                          ? "border-b-2 border-cyan-500 text-cyan-400"
                          : "text-slate-400 hover:text-slate-200"
                      }`}
                    >
                      Targets & Requirements
                    </button>
                  )}
                </div>

                <div className="max-h-60 overflow-y-auto p-3">
                  {attachTab === "files" && files && files.length > 0 && (
                    <FileTree
                      files={files}
                      selectionMode="checkbox"
                      selectedPaths={selectedFilePaths}
                      onSelectionChange={setSelectedFilePaths}
                      className="border-0 bg-transparent p-0"
                    />
                  )}

                  {attachTab === "targets" && hasArchive && (
                    <div className="space-y-3">
                      {archiveTargetList.length > 0 && (
                        <div className="space-y-1">
                          <p className="text-xs font-medium uppercase tracking-wider text-slate-500">Operational Targets</p>
                          {archiveTargetList.map((target) => (
                            <label
                              key={target.id}
                              className="flex items-start gap-2 rounded-md px-2 py-1 text-sm hover:bg-slate-800/50 cursor-pointer"
                            >
                              <input
                                type="checkbox"
                                checked={selectedTargetIds.has(target.id)}
                                onChange={() => {
                                  setSelectedTargetIds((prev) => {
                                    const next = new Set(prev);
                                    if (next.has(target.id)) next.delete(target.id);
                                    else next.add(target.id);
                                    return next;
                                  });
                                }}
                                className="mt-0.5 h-4 w-4 accent-cyan-500"
                              />
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-1.5">
                                  <span className={`text-[10px] font-medium ${
                                    target.criticality === "P0" ? "text-red-400" :
                                    target.criticality === "P1" ? "text-orange-400" : "text-green-400"
                                  }`}>
                                    {target.criticality}
                                  </span>
                                  <span className="font-mono text-xs text-slate-500">{target.id}</span>
                                  {target.status === "complete" ? (
                                    <CheckCircle2 className="h-3 w-3 text-green-400" />
                                  ) : (
                                    <Circle className="h-3 w-3 text-slate-500" />
                                  )}
                                </div>
                                <p className="text-xs text-slate-300">{target.title}</p>
                              </div>
                            </label>
                          ))}
                        </div>
                      )}

                      {archiveRequirements.length > 0 && (
                        <div className="space-y-1">
                          <p className="text-xs font-medium uppercase tracking-wider text-slate-500">Requirements</p>
                          <RequirementCheckboxGroup
                            groups={archiveRequirements}
                            selectedIds={selectedRequirementIds}
                            onToggle={(id) => {
                              setSelectedRequirementIds((prev) => {
                                const next = new Set(prev);
                                if (next.has(id)) next.delete(id);
                                else next.add(id);
                                return next;
                              });
                            }}
                          />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            onClick={() =>
              onSubmit({
                mode: isIdea ? mode : undefined,
                prompt,
                targetKind: isResearch ? targetKind : undefined,
                contextPaths: selectedFilePaths.size > 0 ? [...selectedFilePaths] : undefined,
                contextTargetIds: selectedTargetIds.size > 0 ? [...selectedTargetIds] : undefined,
                contextRequirementIds: selectedRequirementIds.size > 0 ? [...selectedRequirementIds] : undefined,
              })
            }
            disabled={isSubmitting}
            data-testid={selectors.backlogForm.agentSubmit}
          >
            {isSubmitting ? "Spawning..." : `Run ${isIdea ? activeMode?.title ?? "Agent" : "Research"}`}
          </Button>
        </div>
      </div>
    </div>
  );
}
