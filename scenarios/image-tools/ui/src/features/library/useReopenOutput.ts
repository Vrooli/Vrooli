import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import { fetchBlob } from "../../api/client";
import { setWorkspaceIntent } from "../workspace/workspaceIntent";
import type { OutputItem } from "./outputs";

/**
 * Reopen a stored output in the Workspace: fetch its blob, wrap it as a File,
 * hand it off via the Workspace intent, and navigate. Shared by the Home recent
 * rail and the Library grid so "open" behaves identically in both.
 */
export function useReopenOutput(): (item: OutputItem) => Promise<void> {
  const navigate = useNavigate();
  return useCallback(
    async (item: OutputItem) => {
      const blob = await fetchBlob(item.resultRef);
      const ext = item.resultRef.split(".").pop() || "png";
      const file = new File([blob], `library.${ext}`, { type: blob.type || "image/png" });
      setWorkspaceIntent({ file, mode: "edit" });
      navigate("/workspace");
    },
    [navigate],
  );
}
