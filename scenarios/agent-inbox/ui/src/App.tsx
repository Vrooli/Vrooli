import { useState, useCallback, useEffect } from "react";
import { Menu, X, ChevronLeft } from "lucide-react";
import { useChats } from "./hooks/useChats";
import { Sidebar } from "./components/layout/Sidebar";
import { ChatList } from "./components/chat/ChatList";
import { ChatView } from "./components/chat/ChatView";
import { EmptyState } from "./components/chat/EmptyState";
import { LabelManager } from "./components/labels/LabelManager";
import { Button } from "./components/ui/button";

export default function App() {
  const [showLabelManager, setShowLabelManager] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [chatListOpen, setChatListOpen] = useState(true);

  const {
    // State
    selectedChatId,
    currentView,
    isGenerating,
    streamingContent,
    activeToolCalls,

    // Data
    chats,
    chatData,
    models,
    labels,

    // Loading states
    loadingChats,
    loadingChat,

    // Actions
    setCurrentView,
    selectChat,
    sendMessage,

    // Mutations
    createChat,
    deleteChat,
    updateChat,
    toggleRead,
    toggleArchive,
    toggleStar,
    createLabel,
    deleteLabel,
    assignLabel,
    removeLabel,

    // Mutation states
    isCreatingChat,
  } = useChats();

  // Close mobile sidebar when a chat is selected
  useEffect(() => {
    if (selectedChatId && window.innerWidth < 1024) {
      setSidebarOpen(false);
      setChatListOpen(false);
    }
  }, [selectedChatId]);

  // Calculate unread counts for sidebar badges
  const chatCounts = {
    inbox: chats.filter((c) => !c.is_archived).length,
    starred: chats.filter((c) => c.is_starred).length,
    archived: chats.filter((c) => c.is_archived).length,
  };

  const handleSelectChat = useCallback(
    (chatId: string) => {
      selectChat(chatId);
      // Close chat list on mobile when selecting
      if (window.innerWidth < 1024) {
        setChatListOpen(false);
      }
    },
    [selectChat]
  );

  const handleNewChat = useCallback(() => {
    createChat({});
    // Close sidebar on mobile when creating new chat
    if (window.innerWidth < 768) {
      setSidebarOpen(false);
    }
  }, [createChat]);

  const handleBackToList = useCallback(() => {
    setChatListOpen(true);
  }, []);

  return (
    <div className="h-screen bg-slate-950 text-slate-50 flex overflow-hidden" data-testid="inbox-container">
      {/* Mobile Header */}
      <div className="lg:hidden fixed top-0 left-0 right-0 z-40 bg-slate-950 border-b border-white/10 p-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSidebarOpen(true)}
            data-testid="mobile-menu-button"
          >
            <Menu className="h-5 w-5" />
          </Button>
          {selectedChatId && !chatListOpen && (
            <Button variant="ghost" size="icon" onClick={handleBackToList}>
              <ChevronLeft className="h-5 w-5" />
            </Button>
          )}
          <span className="text-sm font-medium">
            {selectedChatId && !chatListOpen
              ? chatData?.chat.name || "Chat"
              : "Agent Inbox"}
          </span>
        </div>
      </div>

      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
          className="lg:hidden fixed inset-0 z-50 bg-black/60 backdrop-blur-sm"
          onClick={() => setSidebarOpen(false)}
          data-testid="mobile-sidebar-overlay"
        />
      )}

      {/* Sidebar - Desktop: always visible, Mobile: slide-in */}
      <div
        className={`fixed lg:relative inset-y-0 left-0 z-50 lg:z-auto transform transition-transform duration-200 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
      >
        <div className="lg:hidden absolute top-3 right-3 z-10">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSidebarOpen(false)}
            data-testid="close-sidebar-button"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>
        <Sidebar
          currentView={currentView}
          onViewChange={(view) => {
            setCurrentView(view);
            if (window.innerWidth < 768) {
              setSidebarOpen(false);
            }
          }}
          onNewChat={handleNewChat}
          onManageLabels={() => setShowLabelManager(true)}
          isCreatingChat={isCreatingChat}
          labels={labels}
          chatCounts={chatCounts}
        />
      </div>

      {/* Chat List - Desktop: always visible, Mobile/Tablet: toggleable */}
      <div
        className={`${
          chatListOpen ? "flex" : "hidden lg:flex"
        } shrink-0 pt-14 lg:pt-0`}
      >
        <ChatList
          chats={chats}
          labels={labels}
          selectedChatId={selectedChatId}
          currentView={currentView}
          isLoading={loadingChats}
          onSelectChat={handleSelectChat}
          onNewChat={handleNewChat}
        />
      </div>

      {/* Main Content - Chat View or Empty State */}
      <div
        className={`flex-1 flex flex-col pt-14 lg:pt-0 ${
          chatListOpen && selectedChatId ? "hidden lg:flex" : "flex"
        }`}
      >
        {selectedChatId ? (
          <ChatView
            chatData={chatData || null}
            models={models}
            labels={labels}
            isLoading={loadingChat}
            isGenerating={isGenerating}
            streamingContent={streamingContent}
            activeToolCalls={activeToolCalls}
            onSendMessage={sendMessage}
            onUpdateChat={(data) => updateChat({ chatId: selectedChatId, data })}
            onToggleRead={() => toggleRead({ chatId: selectedChatId })}
            onToggleStar={() => toggleStar({ chatId: selectedChatId })}
            onToggleArchive={() => toggleArchive({ chatId: selectedChatId })}
            onDeleteChat={() => deleteChat(selectedChatId)}
            onAssignLabel={(labelId) => assignLabel({ chatId: selectedChatId, labelId })}
            onRemoveLabel={(labelId) => removeLabel({ chatId: selectedChatId, labelId })}
          />
        ) : (
          <EmptyState onNewChat={handleNewChat} isCreating={isCreatingChat} />
        )}
      </div>

      {/* Label Manager Dialog */}
      <LabelManager
        open={showLabelManager}
        onClose={() => setShowLabelManager(false)}
        labels={labels}
        onCreateLabel={createLabel}
        onDeleteLabel={deleteLabel}
      />
    </div>
  );
}
