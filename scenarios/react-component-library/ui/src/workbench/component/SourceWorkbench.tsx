import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import {
  beginComponentDraft,
  checkComponentVersion,
  publishComponentDraft,
  updateComponentContent,
  type ComponentVersionCheckResult,
} from "../../api/components";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { wbKeys, type ComponentBundle } from "../data";
import { SourceWorkbenchEditor } from "./SourceWorkbenchEditor";
import { useSourceWorkbenchDocument, type SourceDocument } from "./useSourceWorkbenchDocument";

type DraftCheck = ComponentVersionCheckResult["checks"][number];

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
  onSaved,
}: {
  bundle: ComponentBundle;
  version: string;
  onVersion: (version: string) => void;
  onSaved?: (version: string, filePath: string, sha256: string) => void;
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
  const [bump, setBump] = useState("patch");
  const [checks, setChecks] = useState<DraftCheck[] | null>(null);
  const [checksPassed, setChecksPassed] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [changelog, setChangelog] = useState("");
  const document = useSourceWorkbenchDocument({
    component,
    version: effectiveVersion,
    isDraft,
    entryFilePath: entryFile?.path,
  });
  const {
    filePath,
    editing,
    baselineSha,
    notice,
    setNotice,
    content,
    contentKey,
    contentKeyID,
    activeKeyRef,
    value,
    dirty,
    fileName,
    modelPath,
    reset: resetDocument,
    reload: reloadDocument,
    startEditing: startDocumentEditing,
    discard: discardDocument,
    changeBuffer,
    selectFile,
    clearStashFor,
    markSaved,
    finishPublishing,
  } = document;

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
      const savedBaselineSha = baselineSha;
      const written = await updateComponentContent({
        id: component.id,
        content: savedContent,
        expectedSha256: baselineSha,
        path: savedPath,
      });
      return {
        written,
        savedKeyID,
        savedKey,
        savedVersion,
        savedPath,
        savedContent,
        savedBaselineSha,
      };
    },
    onSuccess: ({
      written,
      savedKeyID,
      savedKey,
      savedVersion,
      savedPath,
      savedContent,
      savedBaselineSha,
    }) => {
      clearStashFor(savedVersion, savedPath);
      queryClient.setQueryData<SourceDocument>(savedKey, {
        content: savedContent,
        sha256: written.sha256,
        sourcePath: written.sourcePath,
      });
      // The person may have switched version or file while the write was in
      // flight. The bytes were saved under the captured key; only apply the
      // editor state when that key is still the one on screen.
      if (activeKeyRef.current !== savedKeyID) return;
      markSaved(written.sha256);
      setChecks(null);
      setChecksPassed(false);
      setNotice(
        t("workbench.component.sourceSaved", {
          defaultValue: "Saved to the draft. Run the version check before publishing.",
        }),
      );
      if (written.sha256 !== savedBaselineSha) onSaved?.(savedVersion, savedPath, written.sha256);
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
      finishPublishing();
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
    resetDocument();
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
    save.reset();
    reloadDocument();
  };

  const startEditing = () => {
    if (!startDocumentEditing()) return;
    setChecks(null);
    setChecksPassed(false);
    save.reset();
    runCheck.reset();
    publish.reset();
  };

  const discard = () => {
    discardDocument();
    save.reset();
    runCheck.reset();
    publish.reset();
  };

  const onBufferChange = (next: string) => {
    changeBuffer(next);
    setChecks(null);
    setChecksPassed(false);
  };

  const mutationError = begin.error ?? save.error ?? runCheck.error ?? publish.error;
  const beginDraft = () => begin.mutate();
  const saveDraft = () => {
    if (!save.isPending) save.mutate();
  };

  return (
    <SourceWorkbenchEditor
      state={{
        version: effectiveVersion,
        versionStatus: isDraft
          ? "draft"
          : selectedRow?.status === ComponentVersionStatus.RELEASED
            ? "released"
            : "indexed",
        dirty,
        sha256: content.data?.sha256 ?? "",
        files,
        entryFilePath: entryFile?.path,
        filePath,
        fileName,
        modelPath,
        isDraft,
        draftVersion,
        canStartDraft,
        bump,
        beginPending: begin.isPending,
        source: content.data,
        loading: content.isLoading,
        error: content.isError,
        editing,
        value,
        savePending: save.isPending,
        notice,
        mutationError: mutationError
          ? { message: describeError(mutationError, t), conflict: isConflict(mutationError) }
          : null,
        draftPanel: {
          state: {
            dirty,
            checks,
            checksPassed,
            runCheckPending: runCheck.isPending,
            publishing,
            publishPending: publish.isPending,
            changelog,
          },
          actions: {
            runChecks: () => runCheck.mutate(),
            preparePublish: () => setPublishing(true),
            cancelPublish: () => setPublishing(false),
            publish: () => publish.mutate(),
            changeChangelog: setChangelog,
          },
        },
      }}
      actions={{
        onVersion,
        onFileSelect: selectFile,
        onBump: setBump,
        onBeginDraft: beginDraft,
        onStartEditing: startEditing,
        onDiscard: discard,
        onSave: saveDraft,
        onBufferChange,
        onReload: reload,
      }}
    />
  );
}
