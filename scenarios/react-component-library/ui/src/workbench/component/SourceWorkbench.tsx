import { Suspense, lazy, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import {
  Check,
  CircleAlert,
  FileCode2,
  GitBranch,
  Pencil,
  Play,
  RotateCcw,
  Save,
  UploadCloud,
  X,
} from "lucide-react";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import {
  beginComponentDraft,
  checkComponentVersion,
  componentsClient,
  publishComponentDraft,
  updateComponentContent,
  type ComponentVersionCheckResult,
} from "../../api/components";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { wbKeys, type ComponentBundle } from "../data";
import { Banner, Chip, Empty, Field, Select, Skeleton, TextArea, WbButton } from "../primitives";

const CodeEditor = lazy(() => import("./CodeEditor"));

interface SourceDoc {
  content: string;
  sha256: string;
  sourcePath: string;
}

type DraftCheck = ComponentVersionCheckResult["checks"][number];

function stashKey(componentID: string, version: string, filePath: string): string {
  return `rcl.workbench.source-draft.${componentID}.${version}.${filePath || "entry"}`;
}

function readStash(componentID: string, version: string, filePath: string): string | null {
  try {
    return window.sessionStorage.getItem(stashKey(componentID, version, filePath));
  } catch {
    return null;
  }
}

function writeStash(componentID: string, version: string, filePath: string, value: string): void {
  try {
    window.sessionStorage.setItem(stashKey(componentID, version, filePath), value);
  } catch {
    /* session-only convenience; the editor still works without it */
  }
}

function clearStash(componentID: string, version: string, filePath: string): void {
  try {
    window.sessionStorage.removeItem(stashKey(componentID, version, filePath));
  } catch {
    /* nothing to clear */
  }
}

function isConflict(error: unknown): boolean {
  return (
    error instanceof ConnectError &&
    error.code === Code.FailedPrecondition &&
    /sha|conflict|changed/i.test(error.rawMessage)
  );
}

function describeError(error: unknown, t: TFunction): string {
  if (isConflict(error)) {
    return t("workbench.component.sourceConflict", {
      defaultValue:
        "The file changed on disk after you opened it. Reload the source before saving again so a concurrent edit is never overwritten.",
    });
  }
  return errorMessage(error, t);
}

/**
 * The governed source surface: read any immutable version, begin or open a
 * mutable draft, edit one artifact with an optimistic-concurrency guard, run
 * the focused version check, then publish. Every mutating call is the same
 * RPC the CLI uses, so the browser cannot bypass draft, check or publish
 * policy — it only removes the copy-commands detour.
 */
export function SourceWorkbench({
  bundle,
  version,
  onVersion,
}: {
  bundle: ComponentBundle;
  version: string;
  onVersion: (version: string) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const component = bundle.asset;

  const draftVersion = useMemo(() => {
    const fromComponent = component.draftVersion.trim();
    if (fromComponent) return fromComponent;
    return (
      bundle.versions.find((row) => row.status === ComponentVersionStatus.DRAFT)?.version ?? ""
    );
  }, [bundle.versions, component.draftVersion]);

  const selectedRow = useMemo(
    () =>
      bundle.versions.find((row) => row.version === version) ??
      bundle.versions.find((row) => row.version === draftVersion) ??
      bundle.versions.find((row) => row.version === component.latestVersion) ??
      bundle.versions[0],
    [bundle.versions, component.latestVersion, draftVersion, version],
  );

  const effectiveVersion = selectedRow?.version ?? version;
  const isDraft = Boolean(draftVersion) && effectiveVersion === draftVersion;
  const isLatest = !component.latestVersion || effectiveVersion === component.latestVersion;
  const canStartDraft = !draftVersion && isLatest;

  const files = selectedRow?.files ?? [];
  const entryFile = files.find((file) => file.isEntry);
  const [filePath, setFilePath] = useState("");
  const [editing, setEditing] = useState(false);
  const [buffer, setBuffer] = useState<string | null>(null);
  const [baselineSha, setBaselineSha] = useState("");
  const [bump, setBump] = useState("patch");
  const [checks, setChecks] = useState<DraftCheck[] | null>(null);
  const [checksPassed, setChecksPassed] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [changelog, setChangelog] = useState("");
  const [notice, setNotice] = useState("");

  const contentKey = [
    "wb",
    "source",
    component.id,
    effectiveVersion,
    isDraft ? "draft" : "release",
    filePath,
  ] as const;
  const contentKeyID = contentKey.join("|");
  const activeKeyRef = useRef(contentKeyID);
  activeKeyRef.current = contentKeyID;

  const content = useQuery({
    queryKey: contentKey,
    queryFn: async (): Promise<SourceDoc> => {
      if (isDraft) {
        // A draft is mutable: read the live file so a save is visible
        // immediately. The indexed snapshot lags until the next check.
        const response = await componentsClient.getComponentContent({
          id: component.id,
          path: filePath,
        });
        return {
          content: response.content,
          sha256: response.sha256,
          sourcePath: response.sourcePath,
        };
      }
      const response = await componentsClient.getComponentVersionContent({
        componentId: component.id,
        version: effectiveVersion,
        path: filePath,
      });
      const file = response.version?.files.find(
        (row) => row.path === (filePath || response.version?.sourcePath),
      );
      return {
        content: response.content,
        sha256: filePath && file ? file.contentSha256 : (response.version?.contentSha256 ?? ""),
        sourcePath: response.version?.sourcePath ?? effectiveVersion,
      };
    },
    staleTime: 15_000,
    retry: 0,
    enabled: Boolean(effectiveVersion && component.id),
  });

  // Restore an unsaved buffer that outlived a tab switch or a route change.
  useEffect(() => {
    if (!content.data) return;
    const stash = isDraft ? readStash(component.id, effectiveVersion, filePath) : null;
    if (stash !== null && stash !== content.data.content) {
      setBuffer(stash);
      setEditing(true);
      setBaselineSha(content.data.sha256);
      setNotice(
        t("workbench.component.sourceRestored", {
          defaultValue: "Restored unsaved changes from this browser session.",
        }),
      );
      return;
    }
    setBuffer(null);
    setBaselineSha(content.data.sha256);
  }, [component.id, content.data, effectiveVersion, filePath, isDraft, t]);

  const value = buffer ?? content.data?.content ?? "";
  const dirty = buffer !== null && buffer !== (content.data?.content ?? "");
  const fileName = filePath || entryFile?.path || `${component.slug || "component"}.tsx`;
  const modelPath = `${component.id}@${effectiveVersion}/${fileName}`;

  const begin = useMutation({
    mutationFn: () => beginComponentDraft({ component: component.libraryId || component.id, bump }),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(component.id) });
      void queryClient.invalidateQueries({ queryKey: wbKeys.catalog });
      if (result.version) onVersion(result.version);
    },
  });

  const save = useMutation({
    mutationFn: async () => {
      // Capture the target with the request so a later render cannot retarget
      // the post-save cache or stash updates.
      const savedKeyID = contentKeyID;
      const savedKey = [...contentKey] as unknown as typeof contentKey;
      const savedVersion = effectiveVersion;
      const savedPath = filePath;
      const savedContent = value;
      const written = await updateComponentContent({
        id: component.id,
        content: savedContent,
        expectedSha256: baselineSha,
        path: savedPath,
      });
      return { written, savedKeyID, savedKey, savedVersion, savedPath, savedContent };
    },
    onSuccess: ({ written, savedKeyID, savedKey, savedVersion, savedPath, savedContent }) => {
      clearStash(component.id, savedVersion, savedPath);
      queryClient.setQueryData<SourceDoc>(savedKey, {
        content: savedContent,
        sha256: written.sha256,
        sourcePath: written.sourcePath,
      });
      // The person may have switched version or file while the write was in
      // flight. The bytes were saved under the captured key; only apply the
      // editor state when that key is still the one on screen.
      if (activeKeyRef.current !== savedKeyID) return;
      setBuffer(null);
      setBaselineSha(written.sha256);
      setChecks(null);
      setChecksPassed(false);
      setNotice(
        t("workbench.component.sourceSaved", {
          defaultValue: "Saved to the draft. Run the version check before publishing.",
        }),
      );
    },
  });

  const runCheck = useMutation({
    mutationFn: async () => {
      const checkedKeyID = contentKeyID;
      const result = await checkComponentVersion({
        component: component.libraryId || component.id,
        version: effectiveVersion,
      });
      return { result, checkedKeyID };
    },
    onSuccess: ({ result, checkedKeyID }) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(component.id) });
      // A check started against one draft must not paint verdicts on another
      // one that the person selected while it ran.
      if (activeKeyRef.current !== checkedKeyID) return;
      setChecks(result.checks);
      setChecksPassed(result.passed);
      if (!result.passed) {
        setPublishing(false);
      }
    },
  });

  const publish = useMutation({
    mutationFn: () =>
      publishComponentDraft({
        component: component.libraryId || component.id,
        draftVersion: effectiveVersion,
        changelogMd: changelog,
      }),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(component.id) });
      void queryClient.invalidateQueries({ queryKey: wbKeys.catalog });
      void queryClient.invalidateQueries({ queryKey: wbKeys.coverage });
      setBuffer(null);
      setEditing(false);
      setPublishing(false);
      setChecks(null);
      setChangelog("");
      if (result.version) onVersion(result.version);
      setNotice(
        t("workbench.component.sourcePublished", {
          defaultValue: "Published {{version}}. The release is now immutable.",
          version: result.version,
        }),
      );
    },
  });

  // A stale failure from a previous version must not follow the person into a
  // different draft or release. Mutations reset with the selected version.
  useEffect(() => {
    setFilePath("");
    setEditing(false);
    setBuffer(null);
    setChecks(null);
    setChecksPassed(false);
    setPublishing(false);
    setNotice("");
    begin.reset();
    save.reset();
    runCheck.reset();
    publish.reset();
    // The mutation objects are stable; listing the reset seams keeps this
    // effect tied to the version change rather than to every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [effectiveVersion]);

  const reload = () => {
    clearStash(component.id, effectiveVersion, filePath);
    setBuffer(null);
    setEditing(false);
    setNotice("");
    void content.refetch();
  };

  const startEditing = () => {
    if (!content.data) return;
    setBuffer(content.data.content);
    setBaselineSha(content.data.sha256);
    setChecks(null);
    setChecksPassed(false);
    setNotice("");
    save.reset();
    runCheck.reset();
    publish.reset();
    setEditing(true);
  };

  const discard = () => {
    clearStash(component.id, effectiveVersion, filePath);
    setBuffer(null);
    setEditing(false);
    setNotice("");
    save.reset();
    runCheck.reset();
    publish.reset();
  };

  const onBufferChange = (next: string) => {
    setBuffer(next);
    setChecks(null);
    setChecksPassed(false);
    setNotice("");
    if (next === content.data?.content) {
      clearStash(component.id, effectiveVersion, filePath);
      return;
    }
    writeStash(component.id, effectiveVersion, filePath, next);
  };

  const selectFile = (next: string) => {
    if (dirty) {
      setNotice(
        t("workbench.component.sourceUnsavedFile", {
          defaultValue: "Save or discard your changes before switching files.",
        }),
      );
      return;
    }
    setFilePath(next);
    setNotice("");
  };

  const statusChip = isDraft ? (
    <Chip tone="amber">{t("workbench.component.draft", { defaultValue: "Draft" })}</Chip>
  ) : (
    <Chip tone="ice">
      {selectedRow?.status === ComponentVersionStatus.RELEASED
        ? t("workbench.component.released", { defaultValue: "Released" })
        : t("workbench.component.indexed", { defaultValue: "Indexed" })}
    </Chip>
  );

  const mutationError = begin.error ?? save.error ?? runCheck.error ?? publish.error;

  return (
    <div className="wb-source">
      <header className="wb-source-head">
        <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
          <span className="wb-caption">
            {t("workbench.component.sourceHeading", {
              defaultValue: "Source · {{version}}",
              version: effectiveVersion,
            })}
          </span>
          {statusChip}
          {dirty ? (
            <Chip tone="amber">
              {t("workbench.component.unsaved", { defaultValue: "Unsaved" })}
            </Chip>
          ) : null}
        </div>
        {content.data?.sha256 ? (
          <span className="wb-tag" title={content.data.sha256}>
            {content.data.sha256.slice(0, 12)}
          </span>
        ) : null}
      </header>

      {files.length > 1 ? (
        <div className="wb-source-filebar">
          <FileCode2 aria-hidden size={14} />
          <Select
            aria-label={t("workbench.component.sourceFile", { defaultValue: "Source file" })}
            value={filePath}
            disabled={dirty || editing}
            onChange={(event) => selectFile(event.target.value)}
          >
            <option value="">{entryFile?.path ?? "entry"}</option>
            {files
              .filter((file) => !file.isEntry)
              .map((file) => (
                <option key={file.path} value={file.path}>
                  {file.path}
                </option>
              ))}
          </Select>
        </div>
      ) : null}

      {!isDraft ? (
        <div className="wb-source-readonly">
          <Banner tone="ice" icon={<GitBranch aria-hidden size={15} />}>
            {draftVersion
              ? t("workbench.component.sourceDraftAvailable", {
                  defaultValue:
                    "Draft {{version}} is active. Released versions stay immutable; edit the draft and publish when its checks pass.",
                  version: draftVersion,
                })
              : t("workbench.component.sourceNoDraft", {
                  defaultValue:
                    "This is an immutable release. Begin a draft to edit its source; the release stays untouched until you publish.",
                })}
          </Banner>
          <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
            {draftVersion ? (
              <WbButton variant="primary" size="sm" onClick={() => onVersion(draftVersion)}>
                <Pencil aria-hidden size={13} />
                {t("workbench.component.openDraft", { defaultValue: "Open draft" })}
              </WbButton>
            ) : canStartDraft ? (
              <>
                <Select
                  aria-label={t("workbench.component.bump", { defaultValue: "Version bump" })}
                  value={bump}
                  onChange={(event) => setBump(event.target.value)}
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
                  disabled={begin.isPending}
                  onClick={() => begin.mutate()}
                >
                  <GitBranch aria-hidden size={13} />
                  {begin.isPending
                    ? t("workbench.component.beginningDraft", { defaultValue: "Beginning draft…" })
                    : t("workbench.component.beginDraft", { defaultValue: "Begin draft" })}
                </WbButton>
              </>
            ) : null}
          </div>
        </div>
      ) : null}

      {content.isLoading ? <Skeleton height={220} /> : null}
      {content.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {t("workbench.component.sourceError", {
            defaultValue: "The source owner did not return this file.",
          })}
        </Banner>
      ) : null}

      {content.data ? (
        editing && isDraft ? (
          <div className="wb-source-editor">
            <Suspense fallback={<Skeleton height={340} />}>
              <CodeEditor
                modelPath={modelPath}
                filePath={fileName}
                value={value}
                readOnly={false}
                height={340}
                onChange={onBufferChange}
                onSave={() => {
                  if (!save.isPending) save.mutate();
                }}
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
            {content.data.content}
          </pre>
        )
      ) : null}

      <div className="wb-row-between" style={{ gap: 8, flexWrap: "wrap" }}>
        <span className="wb-mono wb-small wb-source-meta">
          {content.data?.sourcePath ?? fileName}
          {content.data
            ? ` · ${content.data.content.length} ${content.data.content.length === 1 ? "char" : "chars"}`
            : ""}
        </span>
        <div className="wb-row" style={{ gap: 6, flexWrap: "wrap" }}>
          {isDraft && content.data && !editing ? (
            <WbButton variant="primary" size="sm" onClick={startEditing}>
              <Pencil aria-hidden size={13} />
              {t("workbench.component.editSource", { defaultValue: "Edit source" })}
            </WbButton>
          ) : null}
          {editing ? (
            <>
              <WbButton variant="ghost" size="sm" onClick={discard} disabled={save.isPending}>
                <X aria-hidden size={13} />
                {t("workbench.component.discard", { defaultValue: "Discard" })}
              </WbButton>
              <WbButton
                variant="primary"
                size="sm"
                disabled={!dirty || save.isPending}
                onClick={() => save.mutate()}
              >
                <Save aria-hidden size={13} />
                {save.isPending
                  ? t("workbench.component.saving", { defaultValue: "Saving…" })
                  : t("workbench.component.save", { defaultValue: "Save draft" })}
              </WbButton>
            </>
          ) : null}
        </div>
      </div>

      {notice ? (
        <p className="wb-small wb-muted" role="status">
          {notice}
        </p>
      ) : null}

      {mutationError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {describeError(mutationError, t)}
          {isConflict(mutationError) ? (
            <div style={{ marginTop: 8 }}>
              <WbButton variant="ghost" size="sm" onClick={reload}>
                <RotateCcw aria-hidden size={13} />
                {t("workbench.component.reloadSource", { defaultValue: "Reload source" })}
              </WbButton>
            </div>
          ) : null}
        </Banner>
      ) : null}

      {isDraft ? (
        <div className="wb-source-actions">
          <div className="wb-row-between">
            <span className="wb-caption">
              {t("workbench.component.draftChecks", { defaultValue: "Draft checks" })}
            </span>
            <WbButton
              variant="ghost"
              size="sm"
              disabled={dirty || runCheck.isPending}
              onClick={() => runCheck.mutate()}
            >
              <Play aria-hidden size={12} />
              {runCheck.isPending
                ? t("workbench.component.runningChecks", { defaultValue: "Checking…" })
                : t("workbench.component.runChecks", { defaultValue: "Run checks" })}
            </WbButton>
          </div>
          {dirty ? (
            <p className="wb-small wb-muted-2">
              {t("workbench.component.checkDirtyHint", {
                defaultValue:
                  "Save the draft before checking so the check reads the bytes you can see.",
              })}
            </p>
          ) : null}
          {checks ? (
            <ul className="wb-checklist">
              {checks.map((check, index) => (
                <li key={`${check.stage}-${index}`}>
                  <span
                    className={`wb-checkdot ${check.verdict === "passed" ? "is-pass" : "is-fail"}`}
                  >
                    {check.verdict === "passed" ? (
                      <Check aria-hidden size={11} />
                    ) : (
                      <X aria-hidden size={11} />
                    )}
                  </span>
                  <span style={{ minWidth: 0 }}>
                    <b className="wb-small">{check.stage}</b>
                    <small className="wb-muted-2 wb-small" style={{ display: "block" }}>
                      {check.message}
                      {check.verdict !== "passed" && check.remediation
                        ? ` — ${check.remediation}`
                        : ""}
                    </small>
                  </span>
                </li>
              ))}
            </ul>
          ) : null}
          {checks && checks.length === 0 ? (
            <Empty
              title={t("workbench.component.noChecksTitle", { defaultValue: "No checks declared" })}
            >
              {t("workbench.component.draftNoChecksBody", {
                defaultValue: "The check owner returned no stages for this draft.",
              })}
            </Empty>
          ) : null}

          {checksPassed ? (
            <div className="wb-source-publish">
              <div className="wb-row-between">
                <span className="wb-caption">
                  {t("workbench.component.publishDraft", { defaultValue: "Publish draft" })}
                </span>
                <Chip tone="green">
                  {t("workbench.component.checksPassed", { defaultValue: "Checks passed" })}
                </Chip>
              </div>
              {publishing ? (
                <Field
                  label={t("workbench.component.changelog", { defaultValue: "Changelog" })}
                  htmlFor="rcl-draft-changelog"
                  hint={t("workbench.component.changelogHint", {
                    defaultValue: "Stored with the release. Markdown is preserved.",
                  })}
                >
                  <TextArea
                    id="rcl-draft-changelog"
                    rows={4}
                    value={changelog}
                    placeholder={t("workbench.component.changelogPlaceholder", {
                      defaultValue: "What changed in this version?",
                    })}
                    onChange={(event) => setChangelog(event.target.value)}
                  />
                </Field>
              ) : null}
              <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
                {publishing ? (
                  <>
                    <WbButton variant="ghost" size="sm" onClick={() => setPublishing(false)}>
                      {t("workbench.component.cancel", { defaultValue: "Cancel" })}
                    </WbButton>
                    <WbButton
                      variant="primary"
                      size="sm"
                      disabled={publish.isPending || dirty}
                      onClick={() => publish.mutate()}
                    >
                      <UploadCloud aria-hidden size={13} />
                      {publish.isPending
                        ? t("workbench.component.publishing", { defaultValue: "Publishing…" })
                        : t("workbench.component.publish", { defaultValue: "Publish release" })}
                    </WbButton>
                  </>
                ) : (
                  <WbButton
                    variant="primary"
                    size="sm"
                    disabled={dirty}
                    onClick={() => setPublishing(true)}
                  >
                    <UploadCloud aria-hidden size={13} />
                    {t("workbench.component.preparePublish", { defaultValue: "Publish…" })}
                  </WbButton>
                )}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

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
