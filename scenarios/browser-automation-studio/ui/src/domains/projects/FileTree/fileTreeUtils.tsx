import type { ReactNode } from "react";
import {
  ClipboardCheck,
  FileCode,
  FileText,
  FolderOpen,
  GitBranch,
  Image,
  Zap,
  type LucideIcon,
} from "lucide-react";
import type { FileTreeNode } from './fileTreeTypes';

const FOLDER_TYPE_ICONS: Record<string, { icon: LucideIcon; color: string }> = {
  actions: { icon: Zap, color: "text-yellow-400" },
  flows: { icon: GitBranch, color: "text-blue-400" },
  workflows: { icon: GitBranch, color: "text-blue-400" },
  cases: { icon: ClipboardCheck, color: "text-green-400" },
  tests: { icon: ClipboardCheck, color: "text-green-400" },
  assets: { icon: Image, color: "text-purple-400" },
};

const getFolderIcon = (folderName: string): { icon: LucideIcon; color: string } => {
  const lower = folderName.toLowerCase();
  return FOLDER_TYPE_ICONS[lower] ?? { icon: FolderOpen, color: "text-yellow-500" };
};

export const fileTypeLabelFromPath = (relPath: string): string | null => {
  const normalized = relPath.toLowerCase();
  if (normalized.endsWith(".action.json")) return "Action";
  if (normalized.endsWith(".flow.json")) return "Flow";
  if (normalized.endsWith(".case.json")) return "Case";
  return null;
};

export const fileKindIcon = (node: FileTreeNode): ReactNode => {
  if (node.kind === "folder") {
    const { icon: FolderIcon, color } = getFolderIcon(node.name);
    return <FolderIcon size={16} className={`${color} flex-shrink-0`} />;
  }
  if (node.kind === "workflow_file") {
    return <FileCode size={16} className="text-green-400 flex-shrink-0" />;
  }
  return <FileText size={16} className="text-gray-400 flex-shrink-0" />;
};
