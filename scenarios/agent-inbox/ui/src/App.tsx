import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Archive,
  Star,
  Mail,
  MailOpen,
  Trash2,
  Send,
  MessageSquare,
  Terminal,
  Loader2,
} from "lucide-react";
import { Button } from "./components/ui/button";
import {
  fetchChats,
  fetchChat,
  createChat,
  deleteChat,
  addMessage,
  toggleRead,
  toggleArchive,
  toggleStar,
  type Chat,
  type Message,
} from "./lib/api";

type View = "inbox" | "starred" | "archived";

export default function App() {
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [messageInput, setMessageInput] = useState("");
  const [currentView, setCurrentView] = useState<View>("inbox");

  // Fetch chats based on current view
  const { data: chats = [], isLoading: loadingChats } = useQuery({
    queryKey: ["chats", currentView],
    queryFn: () =>
      fetchChats({
        archived: currentView === "archived",
        starred: currentView === "starred",
      }),
    refetchInterval: 5000,
  });

  // Fetch selected chat with messages
  const { data: chatData, isLoading: loadingChat } = useQuery({
    queryKey: ["chat", selectedChatId],
    queryFn: () => (selectedChatId ? fetchChat(selectedChatId) : null),
    enabled: !!selectedChatId,
  });

  // Mutations
  const createChatMutation = useMutation({
    mutationFn: createChat,
    onSuccess: (newChat) => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(newChat.id);
    },
  });

  const deleteChatMutation = useMutation({
    mutationFn: deleteChat,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(null);
    },
  });

  const addMessageMutation = useMutation({
    mutationFn: ({ chatId, data }: { chatId: string; data: { role: string; content: string } }) =>
      addMessage(chatId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setMessageInput("");
    },
  });

  const toggleReadMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleRead(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    },
  });

  const toggleArchiveMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleArchive(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const toggleStarMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleStar(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  // Mark as read when selecting a chat
  useEffect(() => {
    if (selectedChatId && chatData?.chat && !chatData.chat.is_read) {
      toggleReadMutation.mutate({ chatId: selectedChatId, value: true });
    }
  }, [selectedChatId, chatData?.chat?.is_read]);

  const handleSendMessage = useCallback(() => {
    if (!selectedChatId || !messageInput.trim()) return;
    addMessageMutation.mutate({
      chatId: selectedChatId,
      data: { role: "user", content: messageInput.trim() },
    });
  }, [selectedChatId, messageInput, addMessageMutation]);

  const handleKeyPress = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSendMessage();
      }
    },
    [handleSendMessage]
  );

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    } else if (diffDays === 1) {
      return "Yesterday";
    } else if (diffDays < 7) {
      return date.toLocaleDateString([], { weekday: "short" });
    } else {
      return date.toLocaleDateString([], { month: "short", day: "numeric" });
    }
  };

  return (
    <div className="h-screen bg-slate-950 text-slate-50 flex" data-testid="inbox-container">
      {/* Sidebar */}
      <div className="w-64 border-r border-white/10 flex flex-col">
        <div className="p-4 border-b border-white/10">
          <h1 className="text-lg font-semibold flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Agent Inbox
          </h1>
        </div>

        <div className="p-2">
          <Button
            onClick={() => createChatMutation.mutate({})}
            disabled={createChatMutation.isPending}
            className="w-full justify-start gap-2"
            data-testid="new-chat-button"
          >
            {createChatMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            New Chat
          </Button>
        </div>

        <nav className="flex-1 p-2 space-y-1">
          <button
            onClick={() => setCurrentView("inbox")}
            className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
              currentView === "inbox"
                ? "bg-white/10 text-white"
                : "text-slate-400 hover:text-white hover:bg-white/5"
            }`}
            data-testid="nav-inbox"
          >
            <Mail className="h-4 w-4" />
            Inbox
          </button>
          <button
            onClick={() => setCurrentView("starred")}
            className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
              currentView === "starred"
                ? "bg-white/10 text-white"
                : "text-slate-400 hover:text-white hover:bg-white/5"
            }`}
            data-testid="nav-starred"
          >
            <Star className="h-4 w-4" />
            Starred
          </button>
          <button
            onClick={() => setCurrentView("archived")}
            className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
              currentView === "archived"
                ? "bg-white/10 text-white"
                : "text-slate-400 hover:text-white hover:bg-white/5"
            }`}
            data-testid="nav-archived"
          >
            <Archive className="h-4 w-4" />
            Archived
          </button>
        </nav>
      </div>

      {/* Chat List */}
      <div className="w-80 border-r border-white/10 flex flex-col">
        <div className="p-4 border-b border-white/10">
          <h2 className="text-sm font-medium text-slate-400 capitalize">{currentView}</h2>
        </div>

        <div className="flex-1 overflow-y-auto" data-testid="chat-list">
          {loadingChats ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
            </div>
          ) : chats.length === 0 ? (
            <div className="p-4 text-center text-slate-500 text-sm">
              No chats yet. Create a new chat to get started.
            </div>
          ) : (
            chats.map((chat: Chat) => (
              <div
                key={chat.id}
                onClick={() => setSelectedChatId(chat.id)}
                className={`p-3 border-b border-white/5 cursor-pointer transition-colors ${
                  selectedChatId === chat.id
                    ? "bg-white/10"
                    : "hover:bg-white/5"
                } ${!chat.is_read ? "bg-indigo-500/10" : ""}`}
                data-testid={`chat-item-${chat.id}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      {chat.view_mode === "terminal" ? (
                        <Terminal className="h-4 w-4 text-slate-500 flex-shrink-0" />
                      ) : (
                        <MessageSquare className="h-4 w-4 text-slate-500 flex-shrink-0" />
                      )}
                      <span
                        className={`text-sm truncate ${
                          !chat.is_read ? "font-semibold text-white" : "text-slate-300"
                        }`}
                      >
                        {chat.name}
                      </span>
                      {chat.is_starred && (
                        <Star className="h-3 w-3 text-yellow-500 fill-yellow-500 flex-shrink-0" />
                      )}
                    </div>
                    <p className="text-xs text-slate-500 truncate mt-1">
                      {chat.preview || "No messages yet"}
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className="text-xs text-slate-500">{formatTime(chat.updated_at)}</span>
                    {!chat.is_read && (
                      <span className="w-2 h-2 bg-indigo-500 rounded-full" data-testid="unread-indicator" />
                    )}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Chat View */}
      <div className="flex-1 flex flex-col">
        {selectedChatId && chatData ? (
          <>
            {/* Chat Header */}
            <div className="p-4 border-b border-white/10 flex items-center justify-between">
              <div>
                <h2 className="font-medium">{chatData.chat.name}</h2>
                <p className="text-xs text-slate-500">
                  {chatData.chat.model} • {chatData.chat.view_mode} mode
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleReadMutation.mutate({ chatId: selectedChatId })}
                  title={chatData.chat.is_read ? "Mark as unread" : "Mark as read"}
                  data-testid="toggle-read-button"
                >
                  {chatData.chat.is_read ? (
                    <Mail className="h-4 w-4" />
                  ) : (
                    <MailOpen className="h-4 w-4" />
                  )}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleStarMutation.mutate({ chatId: selectedChatId })}
                  title={chatData.chat.is_starred ? "Unstar" : "Star"}
                  data-testid="toggle-star-button"
                >
                  <Star
                    className={`h-4 w-4 ${
                      chatData.chat.is_starred ? "text-yellow-500 fill-yellow-500" : ""
                    }`}
                  />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleArchiveMutation.mutate({ chatId: selectedChatId })}
                  title={chatData.chat.is_archived ? "Unarchive" : "Archive"}
                  data-testid="toggle-archive-button"
                >
                  <Archive className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (confirm("Delete this chat?")) {
                      deleteChatMutation.mutate(selectedChatId);
                    }
                  }}
                  title="Delete"
                  data-testid="delete-chat-button"
                >
                  <Trash2 className="h-4 w-4 text-red-400" />
                </Button>
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4" data-testid="message-list">
              {loadingChat ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
                </div>
              ) : chatData.messages.length === 0 ? (
                <div className="text-center text-slate-500 text-sm py-8">
                  Start a conversation by typing a message below.
                </div>
              ) : (
                chatData.messages.map((message: Message) => (
                  <div
                    key={message.id}
                    className={`flex ${
                      message.role === "user" ? "justify-end" : "justify-start"
                    }`}
                    data-testid={`message-${message.id}`}
                  >
                    <div
                      className={`max-w-[80%] rounded-2xl px-4 py-2 ${
                        message.role === "user"
                          ? "bg-indigo-600 text-white"
                          : message.role === "assistant"
                          ? "bg-white/10 text-slate-200"
                          : "bg-slate-800 text-slate-400 text-sm italic"
                      }`}
                    >
                      <p className="whitespace-pre-wrap">{message.content}</p>
                      <p className="text-xs opacity-60 mt-1">
                        {new Date(message.created_at).toLocaleTimeString([], {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Message Input */}
            <div className="p-4 border-t border-white/10">
              <div className="flex items-end gap-2">
                <textarea
                  value={messageInput}
                  onChange={(e) => setMessageInput(e.target.value)}
                  onKeyDown={handleKeyPress}
                  placeholder="Type a message..."
                  className="flex-1 bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  rows={1}
                  data-testid="message-input"
                />
                <Button
                  onClick={handleSendMessage}
                  disabled={!messageInput.trim() || addMessageMutation.isPending}
                  className="rounded-xl"
                  data-testid="send-message-button"
                >
                  {addMessageMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-slate-500">
            <div className="text-center">
              <MessageSquare className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>Select a chat or create a new one</p>
              <Button
                onClick={() => createChatMutation.mutate({})}
                className="mt-4"
                variant="outline"
              >
                <Plus className="h-4 w-4 mr-2" />
                New Chat
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
