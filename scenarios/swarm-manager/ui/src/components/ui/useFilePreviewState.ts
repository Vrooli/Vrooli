/**
 * useFilePreviewState hook
 *
 * Encapsulates all state, queries, and mutations for the FilePreview component.
 * Reads the file service from FileServiceContext to stay entity-agnostic.
 *
 * [REQ:REQ-P0-004] File preview state management
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../lib";
import { getContentTypeForFile, getFileType } from "../../lib/file-type-utils";
import { useFileService } from "../../contexts/FileServiceContext";

type FileDraftState = {
  original: string;
  draft: string;
};

export interface UseFilePreviewStateParams {
  filePath: string;
  fileName: string;
  /** Pre-fetched content string. When provided, skips the internal query. */
  externalContent?: string;
  /** When true, disables editing (save/discard/diff). */
  readOnly: boolean;
}

export function useFilePreviewState({
  filePath,
  fileName,
  externalContent,
  readOnly,
}: UseFilePreviewStateParams) {
  const fileService = useFileService();
  const queryClient = useQueryClient();
  const fileType = getFileType(fileName);
  const isImage = fileType === "image";
  const isEditable = fileType !== "image" && !readOnly;

  const contentQueryKey = useMemo(
    () => [...fileService.queryKeyPrefix, "files", filePath, "content"],
    [fileService.queryKeyPrefix, filePath]
  );

  const [fileStateByPath, setFileStateByPath] = useState<Record<string, FileDraftState>>({});
  const [showMobilePath, setShowMobilePath] = useState(false);
  const [copied, setCopied] = useState(false);
  const [markdownView, setMarkdownView] = useState<"rendered" | "raw">("raw");
  const [isDiffMode, setIsDiffMode] = useState(false);

  const fileState = fileStateByPath[filePath];
  const isDirty = Boolean(fileState && fileState.draft !== fileState.original);

  // Reset transient UI state when file changes
  useEffect(() => {
    setShowMobilePath(false);
    setMarkdownView("raw");
    setIsDiffMode(false);
  }, [filePath]);

  // Auto-clear copied indicator
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 1600);
    return () => clearTimeout(timeout);
  }, [copied]);

  // Fetch file content
  const {
    data: fetchedContent,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: contentQueryKey,
    queryFn: () => fileService.getFileContent(filePath),
    enabled: !isImage && externalContent === undefined,
    ...defaultQueryOptions,
    refetchOnWindowFocus: defaultQueryOptions.refetchOnWindowFocus && !isDirty,
  });
  const content = externalContent ?? fetchedContent;

  // Sync fetched content into draft state
  useEffect(() => {
    if (typeof content !== "string") return;
    setFileStateByPath((prev) => {
      const existing = prev[filePath];
      if (!existing) {
        return { ...prev, [filePath]: { original: content, draft: content } };
      }
      if (existing.original === content) return prev;
      if (existing.draft !== existing.original) return prev;
      return { ...prev, [filePath]: { original: content, draft: content } };
    });
  }, [content, filePath]);

  const draftContent = fileState?.draft ?? content ?? "";
  const originalContent = fileState?.original ?? content ?? "";

  // Exit diff mode when there are no changes
  useEffect(() => {
    if (!isDirty) {
      setIsDiffMode(false);
    }
  }, [isDirty]);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async (nextContent: string) =>
      fileService.saveFileContent(
        filePath,
        nextContent,
        getContentTypeForFile(fileName)
      ),
    onSuccess: (_result, nextContent) => {
      setFileStateByPath((prev) => ({
        ...prev,
        [filePath]: { original: nextContent, draft: nextContent },
      }));
      queryClient.setQueryData(contentQueryKey, nextContent);
      queryClient.invalidateQueries({
        queryKey: [...fileService.queryKeyPrefix, "files"],
      });
      if (filePath.endsWith(fileService.protectedFile)) {
        queryClient.invalidateQueries({
          queryKey: [...fileService.queryKeyPrefix],
        });
      }
    },
  });

  // Reset save mutation state on file change
  useEffect(() => {
    saveMutation.reset();
  }, [filePath, saveMutation]);

  const isSaving = saveMutation.isPending;
  const saveErrorMessage =
    saveMutation.error instanceof Error
      ? saveMutation.error.message
      : saveMutation.error
        ? "Unable to save file."
        : "";

  const handleDraftChange = useCallback(
    (nextValue?: string) => {
      const normalized = nextValue ?? "";
      setFileStateByPath((prev) => {
        const existing =
          prev[filePath] ??
          (typeof content === "string"
            ? { original: content, draft: content }
            : { original: "", draft: "" });
        if (existing.draft === normalized) {
          return prev;
        }
        return {
          ...prev,
          [filePath]: { original: existing.original, draft: normalized },
        };
      });
    },
    [content, filePath]
  );

  const handleDiscard = useCallback(() => {
    setFileStateByPath((prev) => {
      const existing = prev[filePath];
      if (existing) {
        return {
          ...prev,
          [filePath]: { original: existing.original, draft: existing.original },
        };
      }
      if (typeof content === "string") {
        return {
          ...prev,
          [filePath]: { original: content, draft: content },
        };
      }
      return prev;
    });
    setIsDiffMode(false);
  }, [content, filePath]);

  const handleSave = useCallback(() => {
    if (!isEditable || !isDirty || isSaving) return;
    saveMutation.mutate(draftContent);
  }, [draftContent, isDirty, isEditable, isSaving, saveMutation]);

  const handleCopyPath = async () => {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(filePath);
        setCopied(true);
        return;
      } catch {
        // Fall through to showing the path if copy fails.
      }
    }
    setShowMobilePath(true);
  };

  // Derived booleans for rendering decisions
  const showMarkdownToggle = fileType === "markdown" && !isDiffMode;
  const showEditor = isEditable && !isDiffMode && (fileType !== "markdown" || markdownView === "raw");
  const showRenderedMarkdown = fileType === "markdown" && markdownView === "rendered" && !isDiffMode;
  const showDiff = isEditable && isDiffMode && !isLoading && !error;
  const canSave = isEditable && isDirty && !isSaving && !isLoading && !error;
  const canDiscard = isEditable && isDirty && !isSaving;
  const canToggleDiff = isEditable && isDirty && !isSaving;

  return {
    // File metadata
    fileType,
    isImage,
    isEditable,

    // Query state
    isLoading,
    error,
    refetch,

    // Draft state
    draftContent,
    originalContent,
    isDirty,

    // Save state
    isSaving,
    saveErrorMessage,

    // UI toggles
    markdownView,
    setMarkdownView,
    isDiffMode,
    setIsDiffMode,
    showMobilePath,
    setShowMobilePath,
    copied,

    // Derived rendering flags
    showMarkdownToggle,
    showEditor,
    showRenderedMarkdown,
    showDiff,
    canSave,
    canDiscard,
    canToggleDiff,

    // Handlers
    handleDraftChange,
    handleDiscard,
    handleSave,
    handleCopyPath,
  };
}
