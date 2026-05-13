import { useCallback, useState } from "react";
import { uploadFile } from "../api/uploads";
import type { GateResult, InputSource } from "../components/terminal/inputGate";

interface UseImageUploadResult {
  uploadAndInject: (file: File | Blob) => Promise<void>;
  uploading: boolean;
  error: string | null;
}

export function useImageUpload(
  sessionId: string,
  submitInput: (data: string, source: InputSource) => GateResult,
): UseImageUploadResult {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const uploadAndInject = useCallback(
    async (file: File | Blob) => {
      setUploading(true);
      setError(null);
      try {
        const path = await uploadFile(sessionId, file);
        submitInput(path + "\n", "upload");
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Upload failed";
        setError(msg);
      } finally {
        setUploading(false);
      }
    },
    [sessionId, submitInput],
  );

  return { uploadAndInject, uploading, error };
}
