// Path helpers for filesystem paths that arrive from the API.
//
// These paths are produced by the server in its *native* form, so the browser
// may receive either POSIX (`/home/me/docs`) or Windows (`C:\Users\me\docs`,
// `\\server\share\docs`) separators depending on where web-console runs. The
// UI therefore cannot assume "/" — every helper here detects the flavour from
// the path itself rather than from the client platform, which would be the
// wrong machine to ask.

const WINDOWS_DRIVE = /^[A-Za-z]:$/;
const WINDOWS_DRIVE_ROOT = /^[A-Za-z]:[\\/]?$/;

/** Whether a path uses Windows separators or a drive/UNC prefix. */
export function isWindowsPath(path: string): boolean {
  if (path.startsWith("\\\\")) return true; // UNC share
  if (/^[A-Za-z]:[\\/]/.test(path)) return true; // drive letter
  return path.includes("\\") && !path.includes("/");
}

/** The separator that matches the path's own flavour. */
export function separatorFor(path: string): string {
  return isWindowsPath(path) ? "\\" : "/";
}

/**
 * Joins a child name onto a directory path, using the parent's separator and
 * without doubling one that is already there (e.g. at a root like "/" or
 * "C:\").
 */
export function joinPath(dir: string, name: string): string {
  if (!dir) return name;
  const sep = separatorFor(dir);
  return /[\\/]$/.test(dir) ? `${dir}${name}` : `${dir}${sep}${name}`;
}

/** The final segment of a path, or the path itself when it has no separator. */
export function basename(path: string): string {
  const trimmed = path.replace(/[\\/]+$/, "");
  if (!trimmed) return path;
  const parts = trimmed.split(/[\\/]/);
  return parts[parts.length - 1] || trimmed;
}

/** One clickable ancestor in a breadcrumb: what to show, and where it goes. */
export interface PathCrumb {
  label: string;
  path: string;
}

/**
 * Splits an absolute path into breadcrumb segments, each carrying the absolute
 * path it navigates to. The first crumb is the root ("/", "C:\", or the UNC
 * share); the last is the path itself.
 *
 * Returns a single crumb for a relative or empty path, since there is no
 * ancestor chain that can be addressed unambiguously.
 */
export function pathCrumbs(path: string): PathCrumb[] {
  if (!path) return [];

  const windows = isWindowsPath(path);
  const sep = windows ? "\\" : "/";

  if (windows) {
    if (path.startsWith("\\\\")) {
      // UNC: \\server\share\a\b — the share is the smallest addressable root.
      const rest = path.slice(2).split(/[\\/]+/).filter(Boolean);
      const [server, share, ...segments] = rest;
      if (!server || !share) return [{ label: path, path }];
      const root = `\\\\${server}\\${share}`;
      return accumulate(root, `\\\\${server}\\${share}`, segments, sep);
    }
    const match = /^([A-Za-z]:)[\\/]?(.*)$/.exec(path);
    if (!match) return [{ label: path, path }];
    const [, drive = "", rest = ""] = match;
    const segments = rest.split(/[\\/]+/).filter(Boolean);
    return accumulate(`${drive}\\`, `${drive}\\`, segments, sep);
  }

  if (!path.startsWith("/")) return [{ label: path, path }];
  const segments = path.split("/").filter(Boolean);
  return accumulate("/", "/", segments, sep);
}

// accumulate builds crumbs from a root plus the segments below it, keeping
// each crumb's path absolute so any of them can be navigated to directly.
function accumulate(rootLabel: string, rootPath: string, segments: string[], sep: string): PathCrumb[] {
  const crumbs: PathCrumb[] = [{ label: rootLabel, path: rootPath }];
  let current = rootPath;
  for (const segment of segments) {
    current = /[\\/]$/.test(current) ? `${current}${segment}` : `${current}${sep}${segment}`;
    crumbs.push({ label: segment, path: current });
  }
  return crumbs;
}

/** Whether a path names a filesystem root, which has no navigable parent. */
export function isRootPath(path: string): boolean {
  return path === "/" || WINDOWS_DRIVE.test(path) || WINDOWS_DRIVE_ROOT.test(path);
}
