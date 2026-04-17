// DOC: docs/guides/workshop-workflow.md
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
import { useEffect, useMemo, useState } from "react";
import { Sparkles, ChevronDown, ChevronRight, Paperclip, CheckCircle2, Circle } from "lucide-react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { FileTree } from "../ui/file-tree";
import { selectors } from "../../consts/selectors";
import {
  BACKLOG_KIND_LABELS,
  type BacklogKind,
  type BacklogFile,
  type ArchiveTargetsResponse,
} from "../../types";
import { RequirementCheckboxGroup } from "./requirement-checkbox";

interface BacklogAgentDialogProps {
  isOpen: boolean;
  isSubmitting?: boolean;
  backlogKind: BacklogKind;
  backlogTitle: string;
  itemStatus?: string;
  errorMessage?: string | null;
  files?: BacklogFile[];
  archiveTargets?: ArchiveTargetsResponse;
  initialSelectedTargetIds?: Set<string>;
  initialSelectedRequirementIds?: Set<string>;
  onClose: () => void;
  onSubmit: (payload: {
    mode?: string;
    prompt: string;
    contextPaths?: string[];
    contextTargetIds?: string[];
    contextRequirementIds?: string[];
  }) => void;
}

const MODE_OPTIONS: Array<{
  value: string;
  title: string;
  description: string;
}> = [
  {
    value: "workshop",
    title: "Workshop",
    description: "Run a workshop round to refine the implementation plan",
  },
  {
    value: "initialize",
    title: "Initialize",
    description: "Bootstrap the item with a plan scaffold and first round",
  },
];

export function BacklogAgentDialog({
  isOpen,
  isSubmitting = false,
  backlogKind,
  backlogTitle,
  itemStatus,
  errorMessage = null,
  files,
  archiveTargets,
  initialSelectedTargetIds,
  initialSelectedRequirementIds,
  onClose,
  onSubmit,
}: BacklogAgentDialogProps) {
  const [prompt, setPrompt] = useState("");
  const [mode, setMode] = useState("workshop");
  const [selectedFilePaths, setSelectedFilePaths] = useState<Set<string>>(new Set());
  const [selectedTargetIds, setSelectedTargetIds] = useState<Set<string>>(new Set());
  const [selectedRequirementIds, setSelectedRequirementIds] = useState<Set<string>>(new Set());
  const [showAttachContext, setShowAttachContext] = useState(false);
  const [attachTab, setAttachTab] = useState<"files" | "targets">("files");

  const isIdea = backlogKind === "idea";

  const totalAttached = selectedFilePaths.size + selectedTargetIds.size + selectedRequirementIds.size;

  const hasArchive = archiveTargets?.has_archive ?? false;
  const archiveTargetList = archiveTargets?.targets ?? [];
  const archiveRequirements = archiveTargets?.requirements ?? [];
  const hasContextOptions = (files && files.length > 0) || hasArchive;

  useEffect(() => {
    if (isOpen) {
      setPrompt("");
      setMode(itemStatus === "backlog" ? "initialize" : "workshop");
      setSelectedFilePaths(new Set());
      const initTargets = initialSelectedTargetIds?.size ? new Set(initialSelectedTargetIds) : new Set<string>();
      const initReqs = initialSelectedRequirementIds?.size ? new Set(initialSelectedRequirementIds) : new Set<string>();
      setSelectedTargetIds(initTargets);
      setSelectedRequirementIds(initReqs);
      const hasInitialSelections = initTargets.size > 0 || initReqs.size > 0;
      setShowAttachContext(hasInitialSelections);
      setAttachTab(hasInitialSelections ? "targets" : "files");
    }
  }, [isOpen, itemStatus, initialSelectedTargetIds, initialSelectedRequirementIds]);

  const filteredModes = useMemo(() => {
    if (itemStatus === "backlog") return MODE_OPTIONS;
    return MODE_OPTIONS.filter((o) => o.value !== "initialize");
  }, [itemStatus]);

  const activeMode = useMemo(() => MODE_OPTIONS.find((option) => option.value === mode), [mode]);

  const title = isIdea ? "Run Idea Agent" : "Run Research Agent";
  const description = isIdea
    ? "This agent will update files inside the idea folder so you can review, edit, and build on the output."
    : "This agent will update files inside the backlog folder with a research summary and supporting notes.";

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-2xl"
      isLoading={isSubmitting}
      testId={selectors.backlogForm.agentDialog}
    >
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

      {filteredModes.length > 0 && (
        <fieldset className="mt-4 space-y-3" data-testid={selectors.backlogForm.agentMode}>
          <legend className="text-sm font-medium text-slate-300">Agent type</legend>
          <div className="grid gap-3 md:grid-cols-3">
            {filteredModes.map((option) => {
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
                </label>
              );
            })}
          </div>
        </fieldset>
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
                                  const next = new Set<string>(prev);
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
                              const next = new Set<string>(prev);
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
              mode: filteredModes.length > 0 ? mode : undefined,
              prompt,
              contextPaths: selectedFilePaths.size > 0 ? [...selectedFilePaths] : undefined,
              contextTargetIds: selectedTargetIds.size > 0 ? [...selectedTargetIds] : undefined,
              contextRequirementIds: selectedRequirementIds.size > 0 ? [...selectedRequirementIds] : undefined,
            })
          }
          disabled={isSubmitting}
          data-testid={selectors.backlogForm.agentSubmit}
        >
          {isSubmitting ? "Spawning..." : `Run ${filteredModes.length > 0 ? activeMode?.title ?? "Agent" : "Research"}`}
        </Button>
      </div>
    </Dialog>
  );
}
