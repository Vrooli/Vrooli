import { Suspense, lazy } from "react";
import type { ComponentProps } from "react";
import { CircleAlert, FileCode2, GitBranch, Pencil, RotateCcw, Save, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { Banner, Chip, Select, Skeleton, WbButton } from "../primitives";
import { DraftReleasePanel } from "./DraftReleasePanel";

const CodeEditor = lazy(() => import("./CodeEditor"));

interface SourceDocument {
  content: string;
  sha256: string;
  sourcePath: string;
}

interface SourceFile {
  path: string;
  isEntry: boolean;
}

interface SourceWorkbenchEditorState {
  version: string;
  versionStatus: "draft" | "released" | "indexed";
  dirty: boolean;
  sha256: string;
  files: SourceFile[];
  entryFilePath?: string;
  filePath: string;
  fileName: string;
  modelPath: string;
  isDraft: boolean;
  draftVersion: string;
  canStartDraft: boolean;
  bump: string;
  beginPending: boolean;
  source?: SourceDocument;
  loading: boolean;
  error: boolean;
  editing: boolean;
  value: string;
  savePending: boolean;
  notice: string;
  mutationError: { message: string; conflict: boolean } | null;
  draftPanel: ComponentProps<typeof DraftReleasePanel>;
}

interface SourceWorkbenchEditorActions {
  onVersion: (version: string) => void;
  onFileSelect: (path: string) => void;
  onBump: (bump: string) => void;
  onBeginDraft: () => void;
  onStartEditing: () => void;
  onDiscard: () => void;
  onSave: () => void;
  onBufferChange: (value: string) => void;
  onReload: () => void;
}

interface SourceWorkbenchEditorProps {
  state: SourceWorkbenchEditorState;
  actions: SourceWorkbenchEditorActions;
}

export function SourceWorkbenchEditor({ state, actions }: SourceWorkbenchEditorProps) {
  const { t } = useTranslation();

  return (
    <div className="wb-source">
      <header className="wb-source-head">
        <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
          <span className="wb-caption">
            {t("workbench.component.sourceHeading", {
              defaultValue: "Source · {{version}}",
              version: state.version,
            })}
          </span>
          {state.versionStatus === "draft" ? (
            <Chip tone="amber">{t("workbench.component.draft", { defaultValue: "Draft" })}</Chip>
          ) : (
            <Chip tone="ice">
              {state.versionStatus === "released"
                ? t("workbench.component.released", { defaultValue: "Released" })
                : t("workbench.component.indexed", { defaultValue: "Indexed" })}
            </Chip>
          )}
          {state.dirty ? (
            <Chip tone="amber">
              {t("workbench.component.unsaved", { defaultValue: "Unsaved" })}
            </Chip>
          ) : null}
        </div>
        {state.sha256 ? (
          <span className="wb-tag" title={state.sha256}>
            {state.sha256.slice(0, 12)}
          </span>
        ) : null}
      </header>

      {state.files.length > 1 ? (
        <div className="wb-source-filebar">
          <FileCode2 aria-hidden size={14} />
          <Select
            aria-label={t("workbench.component.sourceFile", { defaultValue: "Source file" })}
            value={state.filePath}
            disabled={state.dirty || state.editing}
            onChange={(event) => actions.onFileSelect(event.target.value)}
          >
            <option value="">{state.entryFilePath ?? "entry"}</option>
            {state.files
              .filter((file) => !file.isEntry)
              .map((file) => (
                <option key={file.path} value={file.path}>
                  {file.path}
                </option>
              ))}
          </Select>
        </div>
      ) : null}

      {!state.isDraft ? (
        <div className="wb-source-readonly">
          <Banner tone="ice" icon={<GitBranch aria-hidden size={15} />}>
            {state.draftVersion
              ? t("workbench.component.sourceDraftAvailable", {
                  defaultValue:
                    "Draft {{version}} is active. Released versions stay immutable; edit the draft and publish when its checks pass.",
                  version: state.draftVersion,
                })
              : t("workbench.component.sourceNoDraft", {
                  defaultValue:
                    "This is an immutable release. Begin a draft to edit its source; the release stays untouched until you publish.",
                })}
          </Banner>
          <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
            {state.draftVersion ? (
              <WbButton variant="primary" size="sm" onClick={() => actions.onVersion(state.draftVersion)}>
                <Pencil aria-hidden size={13} />
                {t("workbench.component.openDraft", { defaultValue: "Open draft" })}
              </WbButton>
            ) : state.canStartDraft ? (
              <>
                <Select
                  aria-label={t("workbench.component.bump", { defaultValue: "Version bump" })}
                  value={state.bump}
                  onChange={(event) => actions.onBump(event.target.value)}
                  style={{ width: 120 }}
                >
                  <option value="patch">
                    {t("workbench.component.bumpPatch", { defaultValue: "Patch" })}
                  </option>
                  <option value="minor">
                    {t("workbench.component.bumpMinor", { defaultValue: "Minor" })}
                  </option>
                  <option value="major">
                    {t("workbench.component.bumpMajor", { defaultValue: "Major" })}
                  </option>
                </Select>
                <WbButton
                  variant="primary"
                  size="sm"
                  disabled={state.beginPending}
                  onClick={actions.onBeginDraft}
                >
                  <GitBranch aria-hidden size={13} />
                  {state.beginPending
                    ? t("workbench.component.beginningDraft", { defaultValue: "Beginning draft…" })
                    : t("workbench.component.beginDraft", { defaultValue: "Begin draft" })}
                </WbButton>
              </>
            ) : null}
          </div>
        </div>
      ) : null}

      {state.loading ? <Skeleton height={220} /> : null}
      {state.error ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {t("workbench.component.sourceError", {
            defaultValue: "The source owner did not return this file.",
          })}
        </Banner>
      ) : null}

      {state.source ? (
        state.editing && state.isDraft ? (
          <div className="wb-source-editor">
            <Suspense fallback={<Skeleton height={340} />}>
              <CodeEditor
                modelPath={state.modelPath}
                filePath={state.fileName}
                value={state.value}
                readOnly={false}
                height={340}
                onChange={actions.onBufferChange}
                onSave={actions.onSave}
              />
            </Suspense>
          </div>
        ) : (
          <pre
            className="wb-source-code"
            aria-label={t("workbench.component.sourceCode", { defaultValue: "Source code" })}
            role="region"
            // A scrollable source block must be keyboard-reachable; axe's
            // scrollable-region-focusable rule requires this tab stop.
            // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
            tabIndex={0}
          >
            {state.source.content}
          </pre>
        )
      ) : null}

      <div className="wb-row-between" style={{ gap: 8, flexWrap: "wrap" }}>
        <span className="wb-mono wb-small wb-source-meta">
          {state.source?.sourcePath ?? state.fileName}
          {state.source
            ? ` · ${state.source.content.length} ${state.source.content.length === 1 ? "char" : "chars"}`
            : ""}
        </span>
        <div className="wb-row" style={{ gap: 6, flexWrap: "wrap" }}>
          {state.isDraft && state.source && !state.editing ? (
            <WbButton variant="primary" size="sm" onClick={actions.onStartEditing}>
              <Pencil aria-hidden size={13} />
              {t("workbench.component.editSource", { defaultValue: "Edit source" })}
            </WbButton>
          ) : null}
          {state.editing ? (
            <>
              <WbButton variant="ghost" size="sm" onClick={actions.onDiscard} disabled={state.savePending}>
                <X aria-hidden size={13} />
                {t("workbench.component.discard", { defaultValue: "Discard" })}
              </WbButton>
              <WbButton
                variant="primary"
                size="sm"
                disabled={!state.dirty || state.savePending}
                onClick={actions.onSave}
              >
                <Save aria-hidden size={13} />
                {state.savePending
                  ? t("workbench.component.saving", { defaultValue: "Saving…" })
                  : t("workbench.component.save", { defaultValue: "Save draft" })}
              </WbButton>
            </>
          ) : null}
        </div>
      </div>

      {state.notice ? (
        <p className="wb-small wb-muted" role="status">
          {state.notice}
        </p>
      ) : null}

      {state.mutationError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {state.mutationError.message}
          {state.mutationError.conflict ? (
            <div style={{ marginTop: 8 }}>
              <WbButton variant="ghost" size="sm" onClick={actions.onReload}>
                <RotateCcw aria-hidden size={13} />
                {t("workbench.component.reloadSource", { defaultValue: "Reload source" })}
              </WbButton>
            </div>
          ) : null}
        </Banner>
      ) : null}

      {state.isDraft ? <DraftReleasePanel {...state.draftPanel} /> : null}

      <div className="wb-divider" />
      <p className="wb-small wb-muted">
        {t("workbench.component.governedEditBody", {
          defaultValue:
            "A source change goes through a draft, an optimistic-concurrency check, focused validation and an atomic publish, so a browser save can never rewrite a published version.",
        })}
      </p>
    </div>
  );
}
