import { useCallback, useState } from "react";
import { uploadFile } from "../lib/api";

interface UseImageUploadResult {
  uploadAndInject: (file: File | Blob) => Promise<void>;
  uploading: boolean;
  error: string | null;
}

export function useImageUpload(
  sessionId: string,
  sendInput: (data: string) => boolean,
): UseImageUploadResult {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const uploadAndInject = useCallback(
    async (file: File | Blob) => {
      setUploading(true);
      setError(null);
      try {
        const path = await uploadFile(sessionId, file);
        sendInput(path + "\n");
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Upload failed";
        setError(msg);
      } finally {
        setUploading(false);
      }
    },
    [sessionId, sendInput],
  );

  return { uploadAndInject, uploading, error };
}
