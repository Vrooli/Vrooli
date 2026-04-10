import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Truncate a file path from the left, preserving the meaningful end.
 * Always keeps at least the last two segments (parent + basename).
 *
 * @example
 * truncatePath("/home/user/Vrooli/scenarios/web-console", 30)
 * // => "…/scenarios/web-console"
 */
export function truncatePath(path: string, maxLength: number): string {
  if (!path || path.length <= maxLength) return path;

  const segments = path.split("/").filter(Boolean);
  if (segments.length <= 2) return path;

  // Always keep at least the last 2 segments
  let kept = segments.slice(-2);
  let result = `…/${kept.join("/")}`;

  // Try adding more segments from the right while it fits
  for (let i = segments.length - 3; i >= 0; i--) {
    const candidate = `…/${segments.slice(i).join("/")}`;
    if (candidate.length <= maxLength) {
      result = candidate;
    } else {
      break;
    }
  }

  return result;
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Shorten a UUID to its first 8 characters. Non-UUIDs are returned as-is. */
export function shortId(id: string): string {
  if (!id) return id;
  return UUID_RE.test(id) ? id.slice(0, 8) : id;
}

/**
 * Derive a human-friendly display name for a sandbox.
 * Priority: sandbox.name → last 2 segments of scopePath → shortId(id).
 */
export function sandboxDisplayName(sandbox: {
  name?: string;
  scopePath?: string;
  id: string;
}): string {
  if (sandbox.name) return sandbox.name;

  if (sandbox.scopePath) {
    const segments = sandbox.scopePath.split("/").filter(Boolean);
    if (segments.length >= 2) {
      return segments.slice(-2).join("/");
    }
    if (segments.length === 1) {
      return segments[0];
    }
  }

  return shortId(sandbox.id);
}

/**
 * Format an owner identifier for display.
 * UUIDs are shortened to 8 chars. Non-UUIDs are shown as-is.
 * OwnerType is prepended when available (e.g. "agent:407578d1").
 */
export function formatOwner(
  owner: string | undefined,
  ownerType?: string,
): string {
  if (!owner) return "Unknown";
  const display = shortId(owner);
  if (ownerType) return `${ownerType}:${display}`;
  return display;
}

/**
 * Split a file path into directory and filename components.
 *
 * @example
 * splitPath("scenarios/web-console/api/session.go")
 * // => { dir: "scenarios/web-console/api", file: "session.go" }
 */
export function splitPath(filePath: string): { dir: string; file: string } {
  const lastSlash = filePath.lastIndexOf("/");
  if (lastSlash === -1) return { dir: "", file: filePath };
  return {
    dir: filePath.slice(0, lastSlash),
    file: filePath.slice(lastSlash + 1),
  };
}
