import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

export const API_BASE = resolveApiBase({ appendSuffix: true });
export const REPO_HEADER = "X-Repo-Id";

export function buildRepoHeaders(repoId?: string) {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (repoId) {
    headers[REPO_HEADER] = repoId;
  }
  return headers;
}

/** Extract a short readable message from an HTML error page. */
function extractHtmlErrorMessage(html: string, status: number): string {
  const titleMatch = html.match(/<title[^>]*>([\s\S]*?)<\/title>/i);
  if (titleMatch?.[1]) {
    const title = titleMatch[1].trim();
    if (title && title.length < 200) return title;
  }
  const h1Match = html.match(/<h1[^>]*>([\s\S]*?)<\/h1>/i);
  if (h1Match?.[1]) {
    const heading = h1Match[1].replace(/<[^>]*>/g, "").trim();
    if (heading && heading.length < 200) return heading;
  }
  return `Server returned an HTML error (status ${status})`;
}

const MAX_ERROR_LENGTH = 500;

export async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    if (text) {
      // Detect HTML responses (proxies, panic recovery, etc.)
      const isHtml = text.trimStart().startsWith("<") || res.headers.get("content-type")?.includes("text/html");
      if (isHtml) {
        message = extractHtmlErrorMessage(text, res.status);
      } else {
        try {
          const parsed = JSON.parse(text) as { error?: string };
          if (parsed?.error) {
            message = parsed.error;
          }
        } catch {
          // Ignore JSON parse errors; fall back to raw text.
        }
      }
    }
    if (message && message.length > MAX_ERROR_LENGTH) {
      message = message.slice(0, MAX_ERROR_LENGTH) + "\u2026";
    }
    throw new Error(message || `Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function fetchExternalUrl(path: string): Promise<string | null> {
  const res = await fetch(path);
  if (!res.ok) {
    return null;
  }
  const data = await res.json() as { url?: unknown };
  return typeof data.url === "string" && data.url.trim() ? data.url : null;
}

export { buildApiUrl };
