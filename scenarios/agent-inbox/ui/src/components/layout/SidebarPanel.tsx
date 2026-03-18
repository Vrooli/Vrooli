import { forwardRef } from "react";
import { X } from "lucide-react";
import { ErrorBoundary } from "../ErrorBoundary";
import { Sidebar } from "./Sidebar";
import { Button } from "../ui/button";
import type { View } from "../../hooks/useChats";
import type { SettingsTab } from "../settings/settingsTypes";
import type { BulkOperation } from "../../lib/api-chat";
import type { Chat, Label } from "../../lib/api-types";

interface SidebarPanelProps {
  sidebarOpen: boolean;
  chatListOpen: boolean;
  sidebarCollapsed: boolean;
  sidebarWidth: number;
  isSidebarResizing: boolean;
  isMobile: boolean;
  handleResizeStart: (e: React.MouseEvent) => void;
  setSidebarOpen: (open: boolean) => void;
  setChatListOpen: (open: boolean) => void;
  handleToggleSidebarCollapsed: () => void;
  // Sidebar content props
  currentView: View;
  setCurrentView: (view: View) => void;
  handleNewChat: () => void;
  handleNewAgentChat: () => void;
  onManageLabels: () => void;
  onOpenSettings: (tab?: SettingsTab) => void;
  onShowKeyboardShortcuts: () => void;
  isCreatingChat: boolean;
  labels: Label[];
  chatCounts: { inbox: number; starred: number; archived: number };
  chats: Chat[];
  selectedChatId: string | null;
  focusedIndex: number;
  isLoadingChats: boolean;
  onSelectChat: (chatId: string, messageId?: string) => void;
  onRenameChat: (chatId: string, newName: string) => void;
  onBulkOperate: (chatIds: string[], operation: BulkOperation, labelId?: string) => void;
  isBulkOperating: boolean;
  onClearArchived: () => Promise<void>;
  isClearingArchived: boolean;
  onDeselectChat: () => void;
  // Test IDs
  mobileSidebarOverlayTestId: string;
  closeSidebarButtonTestId: string;
}

export const SidebarPanel = forwardRef<HTMLInputElement, SidebarPanelProps>(function SidebarPanel({
  sidebarOpen,
  chatListOpen,
  sidebarCollapsed,
  sidebarWidth,
  isSidebarResizing,
  isMobile,
  handleResizeStart,
  setSidebarOpen,
  setChatListOpen,
  handleToggleSidebarCollapsed,
  currentView,
  setCurrentView,
  handleNewChat,
  handleNewAgentChat,
  onManageLabels,
  onOpenSettings,
  onShowKeyboardShortcuts,
  isCreatingChat,
  labels,
  chatCounts,
  chats,
  selectedChatId,
  focusedIndex,
  isLoadingChats,
  onSelectChat,
  onRenameChat,
  onBulkOperate,
  isBulkOperating,
  onClearArchived,
  isClearingArchived,
  onDeselectChat,
  mobileSidebarOverlayTestId,
  closeSidebarButtonTestId,
}, ref) {
  return (
    <>
      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
          className="lg:hidden fixed inset-0 z-50 bg-black/60 backdrop-blur-sm"
          onClick={() => setSidebarOpen(false)}
          data-testid={mobileSidebarOverlayTestId}
        />
      )}

      {/* Sidebar */}
      <div
        className={`fixed lg:relative inset-y-0 left-0 z-50 lg:z-auto transform transition-transform duration-200 ${
          sidebarOpen || chatListOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        } w-[85vw] max-w-[320px] lg:w-auto lg:max-w-none`}
        style={!isMobile && !(sidebarCollapsed && !isMobile) ? { width: sidebarWidth } : undefined}
      >
        <div className="lg:hidden absolute top-3 right-3 z-10">
          <Button variant="ghost" size="icon" onClick={() => { setSidebarOpen(false); setChatListOpen(false); }} data-testid={closeSidebarButtonTestId}>
            <X className="h-5 w-5" />
          </Button>
        </div>
        <ErrorBoundary name="Sidebar">
          <Sidebar
            ref={ref}
            currentView={currentView}
            onViewChange={(view) => { setCurrentView(view); if (window.innerWidth < 768) setSidebarOpen(false); }}
            onNewChat={handleNewChat}
            onNewAgentChat={handleNewAgentChat}
            onManageLabels={onManageLabels}
            onOpenSettings={onOpenSettings}
            onShowKeyboardShortcuts={onShowKeyboardShortcuts}
            isCreatingChat={isCreatingChat}
            labels={labels}
            chatCounts={chatCounts}
            chats={chats}
            selectedChatId={selectedChatId}
            focusedIndex={focusedIndex}
            isLoadingChats={isLoadingChats}
            onSelectChat={onSelectChat}
            onRenameChat={onRenameChat}
            onBulkOperate={onBulkOperate}
            isBulkOperating={isBulkOperating}
            isCollapsed={sidebarCollapsed && !isMobile}
            onToggleCollapsed={handleToggleSidebarCollapsed}
            onClearArchived={onClearArchived}
            isClearingArchived={isClearingArchived}
            onDeselectChat={onDeselectChat}
          />
        </ErrorBoundary>
      </div>

      {/* Resize Handle - Desktop only */}
      {!isMobile && !(sidebarCollapsed && !isMobile) && (
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize sidebar"
          className={`hidden lg:flex items-center justify-center w-3 cursor-col-resize group shrink-0 ${
            isSidebarResizing ? "bg-indigo-500/10" : "hover:bg-white/5"
          }`}
          onMouseDown={handleResizeStart}
        >
          <div className={`w-px h-8 rounded-full transition-colors ${
            isSidebarResizing ? "bg-indigo-500" : "bg-white/20 group-hover:bg-white/40"
          }`} />
        </div>
      )}
    </>
  );
});
