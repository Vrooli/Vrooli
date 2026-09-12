import type { RowAction } from "@vrooli/react-component-library/useCollection/1";
import type { Skill } from "@/types";

export interface SkillActionHandlers {
  onOpen: (skill: Skill) => void;
  onCopy?: (skill: Skill) => void;
  onMoveToFolder?: (skill: Skill, path: string[]) => void;
  onChangeStorage?: (skill: Skill, folder: Skill["folder"]) => void;
  onCreateNewFolder?: (skill: Skill) => void;
  availableModePaths?: string[][];
}

export function skillActions(handlers: SkillActionHandlers): RowAction<Skill>[] {
  const actions: RowAction<Skill>[] = [
    { id: "open", label: "Open", shortcut: "Enter", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onOpen(skill); } },
  ];
  if (handlers.onCopy) actions.push({ id: "copy", label: "Copy skill", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onCopy?.(skill); } });
  if (handlers.onMoveToFolder) {
    actions.push({ id: "move-root", label: "Move to (Root)", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onMoveToFolder?.(skill, []); } });
    const uniquePaths = new Map((handlers.availableModePaths ?? []).map((path) => [path.join("/"), path]));
    for (const path of uniquePaths.values()) {
      const label = path.join(" / ");
      actions.push({ id: `move-${path.join("-")}`, label: `Move to ${label}`, onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onMoveToFolder?.(skill, path); } });
    }
  }
  if (handlers.onCreateNewFolder) actions.push({ id: "new-folder", label: "Create new folder", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onCreateNewFolder?.(skill); } });
  if (handlers.onChangeStorage) {
    actions.push({ id: "storage-local", label: "Storage: Local", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onChangeStorage?.(skill, "local"); } });
    actions.push({ id: "storage-core", label: "Storage: Core", onSelect: (rows) => { const skill = rows[0]; if (skill) handlers.onChangeStorage?.(skill, "core"); } });
  }
  return actions;
}

export function syncSkillSelection(
  nextKeys: string[],
  currentKeys: Set<string> | undefined,
  toggle: ((id: string) => void) | undefined,
) {
  if (!toggle) return;
  const next = new Set(nextKeys);
  currentKeys?.forEach((id) => { if (!next.has(id)) toggle(id); });
  next.forEach((id) => { if (!currentKeys?.has(id)) toggle(id); });
}
