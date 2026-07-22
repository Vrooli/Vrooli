import type { Ref } from "react";
import { Code, Eye, List } from "lucide-react";
import Editor from "@monaco-editor/react";
import { MarkdownRenderer } from "../markdown/MarkdownRenderer";
import { Button } from "../ui/button";
import { InlineLoadingIndicator } from "../ui/loading-states";
import { selectors } from "../../consts/selectors";
import { formatRelativeTime } from "../../lib";
import type { PromptSkillSummary, PromptSkillVersion } from "../../types";

const formatUsageLabel = (value: string) => value.replace(/_/g, " ");
const joinParts = (parts?: string[]) => (parts && parts.length > 0 ? parts.join(", ") : "-");

const EDITOR_OPTIONS = {
  minimap: { enabled: false },
  wordWrap: "on",
  lineNumbers: "on",
  fontSize: 13,
  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
  tabSize: 2,
  scrollBeyondLastLine: false,
  padding: { top: 12, bottom: 12 },
  renderLineHighlight: "line",
  cursorBlinking: "smooth",
  smoothScrolling: true,
  scrollbar: {
    vertical: "auto",
    horizontal: "auto",
    verticalScrollbarSize: 8,
    horizontalScrollbarSize: 8,
  },
  overviewRulerBorder: false,
  hideCursorInOverviewRuler: true,
  folding: true,
  foldingStrategy: "indentation",
  automaticLayout: true,
} as const;

export interface PromptEditorProps {
  workspaceRef: Ref<HTMLDivElement>;
  skillsPanelRef: Ref<HTMLDivElement>;
  skillsPanelWidth: number;
  isResizing: boolean;
  skillsResizeHandleProps: Record<string, unknown>;
  skillsSidebar: React.ReactNode;
  selectedSkill: PromptSkillSummary | undefined;
  skillLoading: boolean;
  content: string;
  onContentChange: (value: string) => void;
  markdownView: "raw" | "rendered";
  onToggleMarkdownView: () => void;
  markdownPreviewSource: string;
  onSaveDraft: () => void;
  onPublish: () => void;
  updatePending: boolean;
  onShowMobileSkills: () => void;
  versions: PromptSkillVersion[];
  comparisonVersion: PromptSkillVersion | null;
  onCompare: (version: PromptSkillVersion) => void;
  onRollback: (version: number) => void;
  revertPending: boolean;
  diffLines: string[];
}

export function PromptEditor({
  workspaceRef,
  skillsPanelRef,
  skillsPanelWidth,
  isResizing,
  skillsResizeHandleProps,
  skillsSidebar,
  selectedSkill,
  skillLoading,
  content,
  onContentChange,
  markdownView,
  onToggleMarkdownView,
  markdownPreviewSource,
  onSaveDraft,
  onPublish,
  updatePending,
  onShowMobileSkills,
  versions,
  comparisonVersion,
  onCompare,
  onRollback,
  revertPending,
  diffLines,
}: PromptEditorProps) {
  return (
    <div className="h-full overflow-hidden rounded-xl border border-white/10 bg-slate-900/30">
      <div
        ref={workspaceRef}
        className={`flex h-[calc(100dvh-12rem)] flex-col lg:h-[calc(100dvh-15rem)] lg:flex-row ${isResizing ? "select-none" : ""}`}
      >
        <div ref={skillsPanelRef} className="hidden lg:flex lg:flex-col" style={{ width: skillsPanelWidth }}>
          {skillsSidebar}
        </div>
        <div
          className="hidden lg:flex w-2 items-center justify-center border-x border-white/10 bg-slate-900/40 cursor-col-resize"
          {...skillsResizeHandleProps}
        >
          <div className="h-10 w-1 rounded-full bg-slate-700/80" />
        </div>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid={selectors.prompts.editor}>
          {skillLoading ? (
            <InlineLoadingIndicator label="Loading prompt skill..." />
          ) : selectedSkill ? (
            <>
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 bg-slate-800/50 px-3 py-2">
                <div>
                  <p className="font-mono text-sm text-cyan-300">{selectedSkill.id}</p>
                  <p className="text-xs text-slate-400">
                    {formatUsageLabel(selectedSkill.usage_type)} {"\u2022"} {joinParts(selectedSkill.groups)} {"\u2022"}{" "}
                    Updated {selectedSkill.updated_at ? formatRelativeTime(selectedSkill.updated_at) : "unknown"} {"\u2022"}{" "}
                    {selectedSkill.impact_summary}
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="lg:hidden"
                    onClick={onShowMobileSkills}
                  >
                    <List className="mr-1.5 h-4 w-4" />
                    Skills
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    data-testid={selectors.prompts.saveDraft}
                    disabled={updatePending}
                    onClick={onSaveDraft}
                  >
                    Save Draft
                  </Button>
                  <Button
                    size="sm"
                    data-testid={selectors.prompts.publish}
                    disabled={updatePending}
                    onClick={onPublish}
                  >
                    Publish
                  </Button>
                  <button
                    type="button"
                    className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-slate-300/40 text-slate-200 transition-colors hover:bg-slate-900/20 hover:text-white"
                    onClick={onToggleMarkdownView}
                    aria-label={markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
                    title={markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
                  >
                    {markdownView === "rendered" ? <Code className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </div>

              {selectedSkill.required_missing && selectedSkill.required_missing.length > 0 ? (
                <div className="mx-3 mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                  Missing required variables: {selectedSkill.required_missing.join(", ")}
                </div>
              ) : null}

              <div className="mt-3 flex min-h-0 flex-1 flex-col px-3 pb-3">
                <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-slate-700/70 bg-slate-950">
                  {markdownView === "raw" ? (
                    <Editor
                      language="markdown"
                      theme="vs-dark"
                      value={content}
                      onChange={(value) => onContentChange(value ?? "")}
                      options={EDITOR_OPTIONS}
                      height="100%"
                      data-testid={selectors.prompts.contentInput}
                    />
                  ) : (
                    <div className="h-full overflow-auto p-4" data-testid={selectors.prompts.preview}>
                      <MarkdownRenderer content={markdownPreviewSource} className="prose prose-invert max-w-none prose-headings:mb-2 prose-headings:mt-4 prose-p:my-2 prose-pre:bg-slate-900 prose-code:text-cyan-300" />
                    </div>
                  )}
                </div>

                <div className="mt-3 grid gap-3 xl:grid-cols-2">
                  <div className="space-y-2" data-testid={selectors.prompts.versions}>
                    <h4 className="text-sm font-semibold text-slate-100">Version History</h4>
                    <div className="max-h-52 space-y-2 overflow-auto pr-1">
                      {versions.map((version) => (
                        <div key={version.version} className="rounded-md border border-slate-700/60 bg-slate-900/40 p-2">
                          <p className="text-xs text-slate-200">
                            v{version.version} {"\u2022"} {formatRelativeTime(version.updatedAt)}
                          </p>
                          <div className="mt-2 flex gap-2">
                            <Button size="sm" variant="outline" onClick={() => onCompare(version)}>
                              Compare
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={revertPending}
                              onClick={() => onRollback(version.version)}
                            >
                              Rollback
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {comparisonVersion ? (
                    <div className="space-y-2">
                      <h4 className="text-sm font-semibold text-slate-100">Diff vs v{comparisonVersion.version}</h4>
                      <div className="max-h-52 overflow-auto rounded-md border border-slate-700/60 bg-slate-950 p-2">
                        <pre className="whitespace-pre-wrap break-words font-mono text-xs text-slate-200">
                          {diffLines.join("\n")}
                        </pre>
                      </div>
                    </div>
                  ) : (
                    <div className="rounded-md border border-slate-700/60 bg-slate-900/30 p-3 text-xs text-slate-400">
                      Select a version to compare against your current draft.
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : (
            <p className="p-4 text-sm text-slate-400">Select a prompt skill to inspect and edit.</p>
          )}
        </div>
      </div>
    </div>
  );
}
