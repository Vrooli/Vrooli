/**
 * File path utilities for the backlog file browser.
 *
 * Extracted from backlog-file-browser.tsx.
 */

import type { BacklogFile } from "../types";

export function collectMatchingFiles(entries: BacklogFile[], query: string): BacklogFile[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return [];

  const matches: BacklogFile[] = [];
  const walk = (items: BacklogFile[]) => {
    items.forEach((item) => {
      if (item.type === "file") {
        const haystack = `${item.name} ${item.path}`.toLowerCase();
        if (haystack.includes(normalized)) {
          matches.push(item);
        }
      }
      if (item.children && item.children.length > 0) {
        walk(item.children);
      }
    });
  };

  walk(entries);
  return matches;
}

export function getParentPath(path: string): string {
  const slashIndex = path.lastIndexOf("/");
  return slashIndex > -1 ? path.slice(0, slashIndex) : "";
}

export function getBaseName(path: string): string {
  const slashIndex = path.lastIndexOf("/");
  return slashIndex > -1 ? path.slice(slashIndex + 1) : path;
}

export function joinPath(parent: string, name: string): string {
  return parent ? `${parent}/${name}` : name;
}

export function normalizeDestinationPath(value: string): string {
  return value.trim().replace(/\\/g, "/").replace(/^\/+/, "").replace(/\/+$/, "");
}
