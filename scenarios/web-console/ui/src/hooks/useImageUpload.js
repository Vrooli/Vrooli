import { useCallback, useState } from "react";
import { uploadFile } from "../api/uploads";
export function useImageUpload(sessionId, submitInput) {
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState(null);
    const uploadAndInject = useCallback(async (file) => {
        setUploading(true);
        setError(null);
        try {
            const path = await uploadFile(sessionId, file);
            submitInput(path + "\n", "upload");
        }
        catch (err) {
            const msg = err instanceof Error ? err.message : "Upload failed";
            setError(msg);
        }
        finally {
            setUploading(false);
        }
    }, [sessionId, submitInput]);
    return { uploadAndInject, uploading, error };
}
