import { homeRelative } from "../api/client";

/** The in-app page that shows a research-home file, or null outside the home. */
export function filePageHref(path: string, researchHome: string | undefined): string | null {
  const rel = homeRelative(path, researchHome);
  return rel === null ? null : `/files/${rel.split("/").map(encodeURIComponent).join("/")}`;
}

/** The folder part of a path, for resolving a file's relative links. */
export function dirname(path: string): string {
  const at = path.lastIndexOf("/");
  return at <= 0 ? "/" : path.slice(0, at);
}

export function basename(path: string): string {
  return path.slice(path.lastIndexOf("/") + 1);
}

/** A path as the operator reads it: `~research/...` inside the research home, else as it is. */
export function shownPath(path: string, researchHome: string | undefined): string {
  return researchHome && path.startsWith(`${researchHome}/`) ? `~research/${path.slice(researchHome.length + 1)}` : path;
}
