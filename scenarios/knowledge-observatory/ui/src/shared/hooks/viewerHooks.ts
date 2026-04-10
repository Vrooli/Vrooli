// DOC: docs/reference/api-endpoints.md#documentation-viewer
import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { DocResetResponse } from "../services/documentationApi";
import { fetchDocContent, resetDocContent } from "../services/documentationApi";
import { buildDocMetaViewModel } from "../controllers/viewerController";
import { recordActivity } from "../lib/activityStore";

export type DocViewMode = "code" | "preview" | "split";

export function useDocViewer(initialPath?: string | null) {
  const [path, setPath] = useState(initialPath ?? "");
  const [viewMode, setViewMode] = useState<DocViewMode>("preview");
  const [resetResult, setResetResult] = useState<DocResetResponse | null>(null);
  const [resetError, setResetError] = useState("");
  const [isResetting, setIsResetting] = useState(false);

  const query = useQuery({
    queryKey: ["docContent", path],
    queryFn: () => fetchDocContent(path, "raw"),
    enabled: Boolean(path.trim()),
  });

  const meta = useMemo(() => buildDocMetaViewModel(query.data), [query.data]);

  useEffect(() => {
    setResetResult(null);
    setResetError("");
  }, [path]);

  const refresh = useCallback(() => {
    if (!path.trim()) return;
    void query.refetch();
  }, [path, query]);

  const runReset = useCallback(
    async (config: { maxAgeDays: number; keepMinEntries: number; previewOnly: boolean }) => {
      const trimmed = path.trim();
      if (!trimmed) return;
      setIsResetting(true);
      setResetError("");
      try {
        const result = await resetDocContent({
          path: trimmed,
          max_age_days: config.maxAgeDays,
          keep_min_entries: config.keepMinEntries,
          preview_only: config.previewOnly,
        });
        setResetResult(result);
        if (!config.previewOnly) {
          recordActivity({
            type: "doc-reset",
            title: "Document reset applied",
            description: trimmed,
            status: "completed",
          });
          await query.refetch();
        }
      } catch (error) {
        setResetError(error instanceof Error ? error.message : "Reset failed");
      } finally {
        setIsResetting(false);
      }
    },
    [path, query],
  );

  const hasError = Boolean(query.error);
  const errorMessage =
    query.error instanceof Error ? query.error.message : "Unable to load document.";

  return {
    path,
    setPath,
    viewMode,
    setViewMode,
    content: query.data,
    meta,
    isLoading: query.isLoading,
    hasError,
    errorMessage,
    refresh,
    resetResult,
    resetError,
    isResetting,
    runReset,
  };
}
