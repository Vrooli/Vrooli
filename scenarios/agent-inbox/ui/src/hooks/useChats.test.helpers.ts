/**
 * Shared test helpers for useChats tests
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import * as api from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchChats: vi.fn(),
  fetchChat: vi.fn(),
  fetchModels: vi.fn(),
  createChat: vi.fn(),
  deleteChat: vi.fn(),
  deleteAllChats: vi.fn(),
  updateChat: vi.fn(),
  addMessage: vi.fn(),
  toggleRead: vi.fn(),
  toggleArchive: vi.fn(),
  toggleStar: vi.fn(),
  autoNameChat: vi.fn(),
  regenerateMessage: vi.fn(),
  editMessage: vi.fn(),
  selectBranch: vi.fn(),
  bulkOperateChats: vi.fn(),
  forkChat: vi.fn(),
}));

// Mock the useCompletion hook
vi.mock("./useCompletion", () => ({
  useCompletion: vi.fn(),
}));

// Mock the useLabels hook
vi.mock("./useLabels", () => ({
  useLabels: vi.fn(),
}));

// Mock settings
vi.mock("../components/settings/Settings", () => ({
  getDefaultModel: vi.fn(() => "gpt-4"),
}));

// Test data
export const mockChat: api.Chat = {
  id: "chat-1",
  name: "Test Chat",
  preview: "Hello world",
  model: "gpt-4",
  view_mode: "bubble",
  chat_mode: "llm",
  is_read: true,
  is_archived: false,
  is_starred: false,
  label_ids: [],
  web_search_enabled: false,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockMessage: api.Message = {
  id: "msg-1",
  chat_id: "chat-1",
  role: "user",
  content: "Hello",
  sibling_index: 0,
  created_at: "2025-01-01T00:00:00Z",
};

export const mockChatWithMessages: api.ChatWithMessages = {
  chat: mockChat,
  messages: [mockMessage],
  tool_call_records: [],
};

export const mockCompletionState = {
  isGenerating: false,
  streamingContent: "",
  generatedImages: [],
  activeToolCalls: [],
  pendingApprovals: [],
  awaitingApprovals: false,
  runCompletion: vi.fn().mockResolvedValue(undefined),
  resetCompletion: vi.fn(),
  cancelCompletion: vi.fn(),
  approveTool: vi.fn().mockResolvedValue({ success: true }),
  rejectTool: vi.fn().mockResolvedValue(undefined),
};

export const mockLabelsState = {
  labels: [],
  loadingLabels: false,
  labelsError: null,
  createLabel: vi.fn(),
  deleteLabel: vi.fn(),
  assignLabel: vi.fn(),
  removeLabel: vi.fn(),
  isCreatingLabel: false,
  isDeletingLabel: false,
};

export function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}
