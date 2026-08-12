// Multipart file uploads for session attachments. Stays on plain HTTP
// because Connect-RPC binary payloads are not yet wired for this path.
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import { extractAPIError } from "../lib/errors";
const API_BASE = resolveApiBase({ appendSuffix: true });
export async function uploadFile(sessionId, file, filename) {
    const url = buildApiUrl(`/sessions/${sessionId}/upload`, { baseUrl: API_BASE });
    const formData = new FormData();
    formData.append("file", file, filename ?? (file instanceof File ? file.name : "image.png"));
    const res = await fetch(url, { method: "POST", body: formData });
    if (!res.ok)
        throw await extractAPIError(res, "File upload failed");
    const data = (await res.json());
    return data.path;
}
