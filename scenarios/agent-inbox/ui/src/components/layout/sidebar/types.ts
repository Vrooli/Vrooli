import type { LucideIcon } from "lucide-react";
import { Inbox, Star, Archive } from "lucide-react";
import type { View } from "../../../hooks/useChats";
import type { Chat, Label, SearchResult, BulkOperation } from "../../../lib/api";
import type { ChatSearchMode } from "../../../hooks/useSearch";

export interface SidebarProps {
  currentView: View;
  onViewChange: (view: View) => void;
  onNewChat: () => void;
  onNewAgentChat?: () => void;
  onManageLabels: () => void;
  onOpenSettings: () => void;
  onShowKeyboardShortcuts: () => void;
  isCreatingChat: boolean;
  labels: Label[];
  chatCounts?: {
    inbox: number;
    starred: number;
    archived: number;
  };
  // Chat list props
  chats: Chat[];
  selectedChatId: string | null;
  focusedIndex?: number;
  isLoadingChats: boolean;
  onSelectChat: (chatId: string, messageId?: string) => void;
  onRenameChat?: (chatId: string, newName: string) => void;
  // Bulk selection props
  onBulkOperate?: (chatIds: string[], operation: BulkOperation, labelId?: string) => void;
  isBulkOperating?: boolean;
  // Collapsed state (desktop only)
  isCollapsed?: boolean;
  onToggleCollapsed?: () => void;
  // Clear archived chats
  onClearArchived?: () => Promise<void>;
  isClearingArchived?: boolean;
  // Navigation to empty state
  onDeselectChat?: () => void;
}

export interface NavItem {
  id: View;
  label: string;
  icon: LucideIcon;
}

export const navItems: NavItem[] = [
  { id: "inbox", label: "Inbox", icon: Inbox },
  { id: "starred", label: "Starred", icon: Star },
  { id: "archived", label: "Archived", icon: Archive },
];

export const viewLabels: Record<View, { emptyMessage: string }> = {
  inbox: {
    emptyMessage: "No chats yet. Start a new conversation!",
  },
  starred: {
    emptyMessage: "No starred chats. Star important conversations to find them quickly.",
  },
  archived: {
    emptyMessage: "No archived chats.",
  },
};

export type { Chat, Label, SearchResult, BulkOperation, View, ChatSearchMode };
