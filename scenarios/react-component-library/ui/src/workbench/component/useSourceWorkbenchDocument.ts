import { useEffect, useState, useRef } from "react";
import { useQuery } from "@tanstack/react-query";

import { componentsClient, getComponentContent } from "../../api/components";
import { useTranslation } from "../../i18n";

export interface SourceDocument {
  content: string;
  sha256: string;
  sourcePath: string;
}

interface UseSourceWorkbenchDocumentOptions {
  component: { id: string; slug: string; libraryId?: string };
  version: string;
  isDraft: boolean;
  entryFilePath?: string;
}

function stashKey(componentId: string, version: string, filePath: string): string {
  return `rcl.workbench.source-draft.${componentId}.${version}.${filePath || "entry"}`;
}

function readStash(componentId: string, version: string, filePath: string): string | null {
  try {
    return window.sessionStorage.getItem(stashKey(componentId, version, filePath));
  } catch {
    return null;
  }
}

function writeStash(componentId: string, version: string, filePath: string, value: string): void {
  try {
    window.sessionStorage.setItem(stashKey(componentId, version, filePath), value);
  } catch {
    /* session-only convenience; the editor still works without it */
  }
}

function clearStash(componentId: string, version: string, filePath: string): void {
  try {
    window.sessionStorage.removeItem(stashKey(componentId, version, filePath));
  } catch {
    /* nothing to clear */
  }
}

/** Owns the selected source document, its editor buffer, and session draft recovery. */
export function useSourceWorkbenchDocument({
  component,
  version,
  isDraft,
  entryFilePath,
}: UseSourceWorkbenchDocumentOptions) {
  const { t } = useTranslation();
  const [filePath, setFilePath] = useState("");
  const [editing, setEditing] = useState(false);
  const [buffer, setBuffer] = useState<string | null>(null);
  const [baselineSha, setBaselineSha] = useState("");
  const [notice, setNotice] = useState("");

  const contentKey = [
    "wb",
    "source",
    component.id,
    version,
    isDraft ? "draft" : "release",
    filePath,
  ] as const;
  const contentKeyID = contentKey.join("|");
  const activeKeyRef = useRef(contentKeyID);
  activeKeyRef.current = contentKeyID;

  const content = useQuery({
    queryKey: contentKey,
    queryFn: async (): Promise<SourceDocument> => {
      if (isDraft) {
        // Drafts are mutable, so read the live file instead of the indexed snapshot.
        const response = await getComponentContent({ id: component.id, path: filePath });
        return {
          content: response.content,
          sha256: response.sha256,
          sourcePath: response.sourcePath,
        };
      }
      const response = await componentsClient.getComponentVersionContent({
        componentId: component.id,
        version,
        path: filePath,
      });
      const file = response.version?.files.find(
        (row) => row.path === (filePath || response.version?.sourcePath),
      );
      return {
        content: response.content,
        sha256: filePath && file ? file.contentSha256 : (response.version?.contentSha256 ?? ""),
        sourcePath: response.version?.sourcePath ?? version,
      };
    },
    staleTime: 15_000,
    retry: 0,
    enabled: Boolean(version && component.id),
  });

  // Restore an unsaved buffer that outlived a tab switch or a route change.
  useEffect(() => {
    if (!content.data) return;
    const stash = isDraft ? readStash(component.id, version, filePath) : null;
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
  }, [component.id, content.data, filePath, isDraft, t, version]);

  const value = buffer ?? content.data?.content ?? "";
  const dirty = buffer !== null && buffer !== (content.data?.content ?? "");
  const fileName = filePath || entryFilePath || `${component.slug || "component"}.tsx`;
  const modelPath = `${component.id}@${version}/${fileName}`;

  const reset = () => {
    setFilePath("");
    setEditing(false);
    setBuffer(null);
    setNotice("");
  };

  const reload = () => {
    clearStash(component.id, version, filePath);
    setBuffer(null);
    setEditing(false);
    setNotice("");
    void content.refetch();
  };

  const startEditing = () => {
    if (!content.data) return false;
    setBuffer(content.data.content);
    setBaselineSha(content.data.sha256);
    setNotice("");
    setEditing(true);
    return true;
  };

  const discard = () => {
    clearStash(component.id, version, filePath);
    setBuffer(null);
    setEditing(false);
    setNotice("");
  };

  const changeBuffer = (next: string) => {
    setBuffer(next);
    setNotice("");
    if (next === content.data?.content) {
      clearStash(component.id, version, filePath);
      return;
    }
    writeStash(component.id, version, filePath, next);
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

  const clearStashFor = (targetVersion: string, targetPath: string) => {
    clearStash(component.id, targetVersion, targetPath);
  };

  const markSaved = (sha256: string) => {
    setBuffer(null);
    setBaselineSha(sha256);
  };

  const finishPublishing = () => {
    setBuffer(null);
    setEditing(false);
  };

  return {
    filePath,
    editing,
    buffer,
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
    reset,
    reload,
    startEditing,
    discard,
    changeBuffer,
    selectFile,
    clearStashFor,
    markSaved,
    finishPublishing,
  };
}
