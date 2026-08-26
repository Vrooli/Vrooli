/** @vrooliComponentSource data-display.code-block */
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";

import { Button } from "../../components/Button";
import { IconButton } from "../../components/IconButton";
import { Tabs } from "@vrooli/react-component-library/Tabs/1.0.0";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { AdoptionFileTree } from "./AdoptionFileTree";
import { ADOPTION_TEMPLATES } from "./adoptionTemplates";
import { VersionDiffViewer } from "../versions/VersionDiffViewer";
import type { DiffRow } from "../../api/versions";
import { useState, type ReactNode } from "react";
import { Braces, FileCode2, Minus, Plus, RotateCcw, Save, X } from "lucide-react";

export type SourceFilesView = "tree" | "source" | "diff";
type SourceFile = { path: string; isEntry: boolean };
type Comparison = { fromLabel: string; toLabel: string; rows: DiffRow[] };

function formatJSON(value: string, pretty: boolean): string {
  if (!pretty) return value;
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

interface ComponentEditorSourceProps {
  id: string;
  libraryId: string;
  renderable: boolean;
  splitView: boolean;
  activeVersionFiles: SourceFile[];
  selectedFile: string;
  selectedTemplate: string;
  filesView: SourceFilesView;
  comparison?: Comparison | null;
  buffer: string;
  appResolvedTheme: string;
  fontSize: number;
  wordWrap: "on" | "off";
  readOnly: boolean;
  dirty: boolean;
  savePending: boolean;
  contentLoading: boolean;
  handleBeforeMount: (monaco: Monaco) => void;
  handleMount: (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => void;
  onSelectFile: (path: string) => void;
  onSelectTemplate: (template: string) => void;
  onFilesViewChange: (view: SourceFilesView) => void;
  onBufferChange: (value: string) => void;
  onSave: () => void;
  onRevert: () => void;
  onToggleWordWrap: () => void;
  onDecreaseFont: () => void;
  onIncreaseFont: () => void;
  metadataSlot?: ReactNode;
  onCloseComparison?: () => void;
}

export function ComponentEditorSource({
  id,
  libraryId,
  renderable,
  splitView,
  activeVersionFiles,
  selectedFile,
  selectedTemplate,
  filesView,
  comparison,
  buffer,
  appResolvedTheme,
  fontSize,
  wordWrap,
  readOnly,
  dirty,
  savePending,
  contentLoading,
  handleBeforeMount,
  handleMount,
  onSelectFile,
  onSelectTemplate,
  onFilesViewChange,
  onBufferChange,
  onSave,
  onRevert,
  onToggleWordWrap,
  onDecreaseFont,
  onIncreaseFont,
  onCloseComparison,
}: ComponentEditorSourceProps) {
  const { t } = useTranslation();
  const [prettyJSON, setPrettyJSON] = useState(true);
  const isJSON = selectedFile.endsWith(".json");
  const displayedBuffer = isJSON ? formatJSON(buffer, prettyJSON) : buffer;
  return (
    <div
      data-testid={
        renderable ? selectors.components.editor.workspacePane : selectors.assets.hookSource
      }
      data-pane="files"
      className="flex h-full min-h-0 flex-col bg-app-background"
    >
      {splitView && (
        <header className="flex h-control-md shrink-0 items-center border-b border-app-border bg-app-surface px-space-2xs text-xs font-semibold">
          {t(strings.components.editor.files)}
        </header>
      )}
      <div className="flex shrink-0 min-w-0 gap-space-3xs overflow-x-auto border-b border-app-border bg-app-surface px-space-2xs py-space-2xs">
        <IconButton
          data-testid={selectors.components.editor.filesTreeTab}
          aria-label={t("components.editor.fileTree", { defaultValue: "Files" })}
          className={`h-control-compact min-h-control-compact min-w-control-compact shrink-0 ${filesView === "tree" ? "bg-app-primary text-app-primary-foreground" : "border border-app-border bg-app-surface"}`}
          onClick={() => onFilesViewChange("tree")}
        >
          <FileCode2 aria-hidden className="h-icon-compact w-icon-compact" />
        </IconButton>
        <div className="min-w-0 flex-1">
          <Tabs
            items={[
              ...activeVersionFiles.map((file) => ({ id: file.path, label: file.path })),
              ...(comparison
                ? [
                    {
                      id: "__comparison__",
                      label: `${t("components.editor.diffTab", { defaultValue: "Diff" })}: ${comparison.fromLabel} → ${comparison.toLabel}`,
                    },
                  ]
                : []),
            ]}
            active={
              filesView === "diff"
                ? "__comparison__"
                : selectedFile || activeVersionFiles.find((file) => file.isEntry)?.path
            }
            onChange={(next) => {
              if (next === "__comparison__") {
                onFilesViewChange("diff");
                return;
              }
              const file = activeVersionFiles.find((candidate) => candidate.path === next);
              if (file) {
                onFilesViewChange("source");
                onSelectFile(file.isEntry ? "" : file.path);
              }
            }}
            ariaLabel={t("components.editor.fileTabsLabel", { defaultValue: "Component files" })}
            itemTestId={(item) =>
              item === "__comparison__" ? undefined : selectors.components.editor.filesSourceTab
            }
          />
        </div>
        {comparison && (
          <IconButton
            data-testid={selectors.components.editor.filesDiffClose}
            aria-label={t("components.editor.closeComparison", {
              defaultValue: "Close comparison",
            })}
            className="shrink-0"
            onClick={onCloseComparison}
          >
            <X aria-hidden className="h-icon-compact w-icon-compact" />
          </IconButton>
        )}
      </div>
      {filesView === "tree" ? (
        <div className="min-h-0 flex-1 overflow-auto p-space-2xs">
          <AdoptionFileTree
            componentId={id}
            files={activeVersionFiles}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            template={selectedTemplate}
            templates={ADOPTION_TEMPLATES}
            onSelectTemplate={onSelectTemplate}
          />
        </div>
      ) : filesView === "diff" && comparison ? (
        <div className="min-h-0 flex-1 overflow-auto p-space-2xs">
          <VersionDiffViewer rows={comparison.rows} />
        </div>
      ) : (
        <>
          <div className="flex shrink-0 flex-wrap items-center justify-between gap-space-2xs border-b border-app-border bg-app-surface px-space-2xs py-space-2xs">
            <div className="flex items-center gap-space-3xs">
              {isJSON && (
                <IconButton
                  data-testid="components-editor-pretty-json"
                  aria-label={prettyJSON ? "Show compact JSON" : "Pretty-print JSON"}
                  aria-pressed={prettyJSON}
                  onClick={() => setPrettyJSON((current) => !current)}
                  className={`h-control-compact min-h-control-compact min-w-control-compact ${prettyJSON ? "bg-app-primary text-app-primary-foreground" : "border border-app-border bg-app-surface"}`}
                >
                  <Braces aria-hidden className="h-icon-compact w-icon-compact" />
                </IconButton>
              )}
              <IconButton
                data-testid={selectors.components.editor.saveButton}
                aria-label={
                  savePending
                    ? t(strings.components.editor.saving)
                    : t(strings.components.editor.save)
                }
                onClick={onSave}
                disabled={readOnly || !dirty || savePending || contentLoading}
                className="h-control-compact min-h-control-compact min-w-control-compact bg-app-primary text-app-primary-foreground"
              >
                <Save aria-hidden className="h-icon-compact w-icon-compact" />
              </IconButton>
              <IconButton
                data-testid={selectors.components.editor.filesRevertButton}
                aria-label={t("components.editor.revert", { defaultValue: "Revert" })}
                onClick={onRevert}
                disabled={readOnly || !dirty}
                className="h-control-compact min-h-control-compact min-w-control-compact border border-app-border bg-app-surface"
              >
                <RotateCcw aria-hidden className="h-icon-compact w-icon-compact" />
              </IconButton>
            </div>
            <div className="flex items-center gap-space-3xs">
              <IconButton
                data-testid={selectors.components.editor.filesWrapButton}
                aria-label={t("components.editor.wrap", { defaultValue: "Wrap" })}
                aria-pressed={wordWrap === "on"}
                onClick={onToggleWordWrap}
                className={`h-control-compact min-h-control-compact min-w-control-compact ${wordWrap === "on" ? "bg-app-primary text-app-primary-foreground" : "border border-app-border bg-app-surface"}`}
              >
                <FileCode2 aria-hidden className="h-icon-compact w-icon-compact" />
              </IconButton>
              <Button
                data-testid={selectors.components.editor.filesFontDecrease}
                type="button"
                variant="secondary"
                aria-label={t("components.editor.decreaseFont", {
                  defaultValue: "Decrease font size",
                })}
                onClick={onDecreaseFont}
                disabled={fontSize <= 11}
                className="h-control-compact w-control-compact p-0"
              >
                <Minus aria-hidden className="h-icon-compact w-icon-compact" />
              </Button>
              <Button
                data-testid={selectors.components.editor.filesFontIncrease}
                type="button"
                variant="secondary"
                aria-label={t("components.editor.increaseFont", {
                  defaultValue: "Increase font size",
                })}
                onClick={onIncreaseFont}
                disabled={fontSize >= 20}
                className="h-control-compact w-control-compact p-0"
              >
                <Plus aria-hidden className="h-icon-compact w-icon-compact" />
              </Button>
            </div>
          </div>
          <div
            data-testid={selectors.components.editor.surface}
            className="min-h-0 flex-1 overflow-hidden"
          >
            <Editor
              height="100%"
              language={isJSON ? "json" : "typescript"}
              path={selectedFile || `${libraryId || id}.tsx`}
              value={displayedBuffer}
              onChange={(value) => onBufferChange(value ?? "")}
              beforeMount={handleBeforeMount}
              onMount={handleMount}
              theme={appResolvedTheme === "dark" ? "vs-dark" : "vs"}
              options={{
                fontSize,
                lineNumbers: "on",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                tabSize: 2,
                insertSpaces: true,
                wordWrap,
                automaticLayout: true,
                readOnly,
              }}
            />
          </div>
        </>
      )}
    </div>
  );
}
