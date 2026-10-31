import type { ArchiveEntry, ArchiveView, PreviewArchive } from "../../api/filePreview";
import { strings } from "../../consts/strings";

const c = strings.archivePreview;
const STATUS_KEYS = {
  ready: c.ready, directory: c.folder, unsafe: c.unsafe, link: c.link, special: c.special,
  encrypted: c.encrypted, unsupported: c.unsupported, too_large: c.tooLarge,
  corrupt: c.corrupt, limit: c.limit, busy: c.busy, timeout: c.timeout, expired: c.expired,
  stale: c.stale, depth: c.depth, missing: c.missing, blocked: c.blocked,
} as const;

export function archiveStatusKey(status: string) {
  return Object.prototype.hasOwnProperty.call(STATUS_KEYS, status)
    ? STATUS_KEYS[status as keyof typeof STATUS_KEYS] : c.corrupt;
}

export interface ArchiveRow extends ArchiveEntry {
  folder?: string;
  fullName: string;
  collisions: number;
  collisionIndex: number;
}

/** Names remain virtual slash paths. Unsafe entries are inert, visible at root. */
export function archiveRows(archive: PreviewArchive, view: ArchiveView): ArchiveRow[] {
  const prefix = view.folder ? `${view.folder}/` : "";
  const folders = new Map<string, ArchiveRow>(), files: ArchiveRow[] = [];
  const collisions = new Map<string, number>(), seen = new Map<string, number>();
  for (const entry of archive.entries) {
    const key = entry.name.toLocaleLowerCase();
    collisions.set(key, (collisions.get(key) ?? 0) + 1);
  }
  for (const entry of archive.entries) {
    const key = entry.name.toLocaleLowerCase(), collisionIndex = (seen.get(key) ?? 0) + 1;
    seen.set(key, collisionIndex);
    const identity = { fullName: entry.name, collisionIndex, collisions: collisions.get(key) ?? 1 };
    if (entry.status === "unsafe") {
      if (!prefix) files.push({ ...entry, ...identity });
      continue;
    }
    if (!entry.name.startsWith(prefix)) continue;
    const segments = entry.name.slice(prefix.length).split("/"), name = segments[0];
    if (!name || (!view.showHidden && name.startsWith("."))) continue;
    if (segments.length > 1 || entry.status === "directory") {
      const folder = prefix + name;
      if (!folders.has(folder)) folders.set(folder, {
        id: `folder:${folder}`, name, status: "directory", sizeBytes: 0,
        folder, collisions: 1, collisionIndex: 1, fullName: folder,
      });
    } else files.push({ ...entry, ...identity, name });
  }
  const query = view.query.trim().toLocaleLowerCase();
  return [...folders.values(), ...files]
    .filter(entry => !query || entry.name.toLocaleLowerCase().includes(query))
    .sort((a, b) => {
      if (Boolean(a.folder) !== Boolean(b.folder)) return a.folder ? -1 : 1;
      if (view.sort === "size" && a.sizeBytes !== b.sizeBytes) return b.sizeBytes - a.sizeBytes;
      return a.name.localeCompare(b.name, undefined, { numeric: true })
        || a.id.localeCompare(b.id, undefined, { numeric: true });
    });
}

// Replace invisible control/spoof characters in the displayed name. The server
// retains the original identity and blocks opening these entries.
export function archiveDisplayName(name: string): string {
  return name.replace(/[\p{Cc}\p{Cf}]/gu, "�");
}
