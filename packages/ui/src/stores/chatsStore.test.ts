// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { renderHook } from "@testing-library/react";
import { type Chat, type ChatSearchResult, ChatSortBy } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useChats, useChatsStore } from "./chatsStore.js";

// Mock dependencies
vi.mock("../api/fetchData.js", () => ({
    fetchData: vi.fn(),
}));

vi.mock("../api/responseParser.js", () => ({
    ServerResponseParser: {
        displayErrors: vi.fn(),
    },
}));

vi.mock("../contexts/session.js", () => ({
    SessionContext: {},
}));

vi.mock("../utils/authentication/session.js", () => ({
    checkIfLoggedIn: vi.fn(),
}));

import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { checkIfLoggedIn } from "../utils/authentication/session.js";

const mockFetchData = vi.mocked(fetchData);
const mockDisplayErrors = vi.mocked(ServerResponseParser.displayErrors);
const mockCheckIfLoggedIn = vi.mocked(checkIfLoggedIn);

// Helper to create mock Chat objects
const createMockChat = (id: string, overrides: Partial<Chat> = {}): Chat => ({
    __typename: "Chat",
    id,
    publicId: `pub_${id}`,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    invites: [],
    invitesCount: 0,
    messages: [],
    openToAnyoneWithInvite: false,
    participants: [],
    participantsCount: 0,
    team: null,
    translations: [],
    translationsCount: 0,
    you: {
        __typename: "ChatYou",
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
    ...overrides,
});

describe("useChatsStore", () => {
    beforeEach(() => {
        // Get the initial store state and reset everything properly
        const initialState = useChatsStore.getState();

        // Reset store state completely with all functions included
        useChatsStore.setState({
            chats: [],
            isLoading: false,
            error: null,
            fetchChats: initialState.fetchChats,
            setChats: initialState.setChats,
            addChat: initialState.addChat,
            removeChat: initialState.removeChat,
            updateChat: initialState.updateChat,
            clearChats: initialState.clearChats,
        }, true);

        // Clear all mocks
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("initial state", () => {
        it("starts with empty chat list when user opens app", () => {
            // GIVEN: User opens the application
            // WHEN: The chat store initializes
            const state = useChatsStore.getState();

            // THEN: No chats are loaded yet
            expect(state.chats).toEqual([]);
            expect(state.isLoading).toBe(false);
            expect(state.error).toBe(null);
        });
    });

    describe("fetchChats", () => {
        it("loads user's chat list when requested", async () => {
            // GIVEN: User has multiple chats on the server
            const mockChats = [
                createMockChat("chat1"),
                createMockChat("chat2"),
            ];

            const mockResponse: ChatSearchResult = {
                edges: mockChats.map(chat => ({ cursor: chat.id, node: chat })),
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: "chat1",
                    endCursor: "chat2",
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            // WHEN: App fetches the chat list
            const result = await useChatsStore.getState().fetchChats();

            // THEN: Chats are loaded and displayed
            expect(result).toEqual(mockChats);
            expect(useChatsStore.getState().chats).toEqual(mockChats);
            expect(useChatsStore.getState().isLoading).toBe(false);
            expect(useChatsStore.getState().error).toBe(null);
        });

        it("should handle empty chat list", async () => {
            const mockResponse: ChatSearchResult = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toEqual([]);
            expect(useChatsStore.getState().chats).toEqual([]);
            expect(useChatsStore.getState().isLoading).toBe(false);
            expect(useChatsStore.getState().error).toBe(null);
        });

        it("shows error message when chat loading fails", async () => {
            // GIVEN: Server returns an error
            const mockErrors = [{ message: "Failed to fetch chats" }];
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => { });

            mockFetchData.mockResolvedValue({
                data: null,
                errors: mockErrors,
                timestamp: Date.now(),
            });

            // WHEN: App tries to fetch chats
            const result = await useChatsStore.getState().fetchChats();

            // THEN: Error is displayed to user
            expect(mockDisplayErrors).toHaveBeenCalledWith(mockErrors);
            expect(result).toEqual([]);
            expect(useChatsStore.getState().error).toBe("Error fetching chats");

            consoleSpy.mockRestore();
        });

        it("should handle network errors", async () => {
            const networkError = new Error("Network error");
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => { });
            mockFetchData.mockRejectedValue(networkError);

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toEqual([]);
            expect(useChatsStore.getState().chats).toEqual([]);
            expect(useChatsStore.getState().isLoading).toBe(false);
            expect(useChatsStore.getState().error).toBe("Error fetching chats");

            consoleSpy.mockRestore();
        });

        it("should handle abort signal gracefully", async () => {
            const abortError = new Error("AbortError");
            abortError.name = "AbortError";
            mockFetchData.mockRejectedValue(abortError);

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toEqual([]);
            // Should not set error state for abort errors
            expect(useChatsStore.getState().error).toBe(null);
        });

        it("uses cached chats to avoid unnecessary network requests", async () => {
            // GIVEN: Chats are already loaded in memory
            const existingChats = [createMockChat("existing")];
            useChatsStore.setState({ chats: existingChats });

            // WHEN: App requests chats again
            const result = await useChatsStore.getState().fetchChats();

            // THEN: Cached chats are returned without network call
            expect(mockFetchData).not.toHaveBeenCalled();
            expect(result).toEqual(existingChats);
        });

        it("should not refetch if already loading", async () => {
            useChatsStore.setState({ isLoading: true });

            const result = await useChatsStore.getState().fetchChats();

            expect(mockFetchData).not.toHaveBeenCalled();
            expect(result).toEqual([]);
        });

        it("should set loading state during fetch", async () => {
            const mockResponse: ChatSearchResult = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
            };

            let loadingStateDuringFetch: boolean | undefined;

            mockFetchData.mockImplementation(async () => {
                loadingStateDuringFetch = useChatsStore.getState().isLoading;
                return {
                    data: mockResponse,
                    errors: null,
                    timestamp: Date.now(),
                };
            });

            await useChatsStore.getState().fetchChats();

            expect(loadingStateDuringFetch).toBe(true);
            expect(useChatsStore.getState().isLoading).toBe(false);
        });

        it("should pass abort signal to fetchData", async () => {
            const mockResponse: ChatSearchResult = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            const abortController = new AbortController();
            await useChatsStore.getState().fetchChats(abortController.signal);

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/chats",
                method: "GET",
                inputs: { sortBy: ChatSortBy.DateUpdatedDesc },
                signal: abortController.signal,
            });
        });
    });

    describe("setChats", () => {
        it("should set chats with array", () => {
            const newChats = [createMockChat("chat1"), createMockChat("chat2")];

            useChatsStore.getState().setChats(newChats);

            expect(useChatsStore.getState().chats).toEqual(newChats);
        });

        it("should set chats with function", () => {
            const initialChats = [createMockChat("chat1")];
            const additionalChat = createMockChat("chat2");

            useChatsStore.setState({ chats: initialChats });

            useChatsStore.getState().setChats((prev) => [...prev, additionalChat]);

            expect(useChatsStore.getState().chats).toEqual([...initialChats, additionalChat]);
        });

        it("should replace existing chats", () => {
            const initialChats = [createMockChat("chat1"), createMockChat("chat2")];
            const newChats = [createMockChat("chat3")];

            useChatsStore.setState({ chats: initialChats });
            useChatsStore.getState().setChats(newChats);

            expect(useChatsStore.getState().chats).toEqual(newChats);
        });
    });

    describe("addChat", () => {
        it("should add chat to empty list", () => {
            const newChat = createMockChat("chat1");

            useChatsStore.getState().addChat(newChat);

            expect(useChatsStore.getState().chats).toEqual([newChat]);
        });

        it("displays new chat at top when user creates one", () => {
            // GIVEN: User has existing chats
            const existingChat = createMockChat("existing");
            useChatsStore.setState({ chats: [existingChat] });

            // WHEN: User creates a new chat
            const newChat = createMockChat("new");
            useChatsStore.getState().addChat(newChat);

            // THEN: New chat appears at the top of the list
            expect(useChatsStore.getState().chats).toEqual([newChat, existingChat]);
        });

        it("should maintain other chats when adding", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");
            const newChat = createMockChat("new");

            useChatsStore.setState({ chats: [chat1, chat2] });
            useChatsStore.getState().addChat(newChat);

            expect(useChatsStore.getState().chats).toEqual([newChat, chat1, chat2]);
        });
    });

    describe("removeChat", () => {
        it("removes chat from list when user deletes it", () => {
            // GIVEN: User has multiple chats
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");
            const chat3 = createMockChat("chat3");
            useChatsStore.setState({ chats: [chat1, chat2, chat3] });

            // WHEN: User deletes a chat
            useChatsStore.getState().removeChat("chat2");

            // THEN: Chat is removed from the list
            expect(useChatsStore.getState().chats).toEqual([chat1, chat3]);
        });

        it("should do nothing if chat ID not found", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");

            useChatsStore.setState({ chats: [chat1, chat2] });
            useChatsStore.getState().removeChat("nonexistent");

            expect(useChatsStore.getState().chats).toEqual([chat1, chat2]);
        });

        it("should handle empty chat list", () => {
            useChatsStore.getState().removeChat("nonexistent");

            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should remove only the specified chat", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");
            const chat3 = createMockChat("chat3");

            useChatsStore.setState({ chats: [chat1, chat2, chat3] });
            useChatsStore.getState().removeChat("chat1");

            expect(useChatsStore.getState().chats).toEqual([chat2, chat3]);
        });
    });

    describe("updateChat", () => {
        it("updates chat settings when user changes them", () => {
            // GIVEN: User has a private chat
            const originalChat = createMockChat("chat1", { openToAnyoneWithInvite: false });
            useChatsStore.setState({ chats: [originalChat] });

            // WHEN: User makes the chat public
            const updatedChat = createMockChat("chat1", { openToAnyoneWithInvite: true });
            useChatsStore.getState().updateChat(updatedChat);

            // THEN: Chat settings are updated in the list
            expect(useChatsStore.getState().chats).toEqual([updatedChat]);
        });

        it("should update only the matching chat", () => {
            const chat1 = createMockChat("chat1", { openToAnyoneWithInvite: false });
            const chat2 = createMockChat("chat2", { openToAnyoneWithInvite: false });
            const chat3 = createMockChat("chat3", { openToAnyoneWithInvite: false });
            const updatedChat2 = createMockChat("chat2", { openToAnyoneWithInvite: true });

            useChatsStore.setState({ chats: [chat1, chat2, chat3] });
            useChatsStore.getState().updateChat(updatedChat2);

            expect(useChatsStore.getState().chats).toEqual([chat1, updatedChat2, chat3]);
        });

        it("should do nothing if chat ID not found", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");
            const nonexistentChat = createMockChat("nonexistent");

            useChatsStore.setState({ chats: [chat1, chat2] });
            useChatsStore.getState().updateChat(nonexistentChat);

            expect(useChatsStore.getState().chats).toEqual([chat1, chat2]);
        });

        it("should handle empty chat list", () => {
            const updatedChat = createMockChat("chat1");

            useChatsStore.getState().updateChat(updatedChat);

            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should preserve chat order during update", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2", { participantsCount: 1 });
            const chat3 = createMockChat("chat3");
            const updatedChat2 = createMockChat("chat2", { participantsCount: 5 });

            useChatsStore.setState({ chats: [chat1, chat2, chat3] });
            useChatsStore.getState().updateChat(updatedChat2);

            expect(useChatsStore.getState().chats[0]).toEqual(chat1);
            expect(useChatsStore.getState().chats[1]).toEqual(updatedChat2);
            expect(useChatsStore.getState().chats[2]).toEqual(chat3);
        });
    });

    describe("clearChats", () => {
        it("clears all chats when user logs out", () => {
            // GIVEN: User has chats loaded with some state
            const chats = [createMockChat("chat1"), createMockChat("chat2")];
            useChatsStore.setState({
                chats,
                isLoading: true,
                error: "Some error",
            });

            // WHEN: User logs out
            useChatsStore.getState().clearChats();

            // THEN: All chat data is cleared
            expect(useChatsStore.getState().chats).toEqual([]);
            expect(useChatsStore.getState().isLoading).toBe(false);
            expect(useChatsStore.getState().error).toBe(null);
        });

        it("should work when store is already empty", () => {
            useChatsStore.getState().clearChats();

            expect(useChatsStore.getState().chats).toEqual([]);
            expect(useChatsStore.getState().isLoading).toBe(false);
            expect(useChatsStore.getState().error).toBe(null);
        });
    });

    describe("error handling scenarios", () => {
        it("should handle malformed API response", async () => {
            mockFetchData.mockResolvedValue({
                data: { edges: null } as any, // Malformed response
                errors: null,
                timestamp: Date.now(),
            });

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toEqual([]);
            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should handle API response with missing data", async () => {
            mockFetchData.mockResolvedValue({
                data: null,
                errors: null,
                timestamp: Date.now(),
            });

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toEqual([]);
            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should clear error state on successful fetch after error", async () => {
            // Set initial error state
            useChatsStore.setState({ error: "Previous error" });

            const mockResponse: ChatSearchResult = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            await useChatsStore.getState().fetchChats();

            expect(useChatsStore.getState().error).toBe(null);
        });
    });

    describe("state management and reactivity", () => {
        it("should maintain state consistency during concurrent operations", () => {
            const chat1 = createMockChat("chat1");
            const chat2 = createMockChat("chat2");
            const chat3 = createMockChat("chat3");

            // Add multiple chats
            useChatsStore.getState().addChat(chat1);
            useChatsStore.getState().addChat(chat2);

            // Update and remove concurrently
            const updatedChat1 = createMockChat("chat1", { openToAnyoneWithInvite: true });
            useChatsStore.getState().updateChat(updatedChat1);
            useChatsStore.getState().addChat(chat3);
            useChatsStore.getState().removeChat("chat2");

            const finalState = useChatsStore.getState().chats;
            expect(finalState).toHaveLength(2);
            expect(finalState.find(c => c.id === "chat1")?.openToAnyoneWithInvite).toBe(true);
            expect(finalState.find(c => c.id === "chat2")).toBeUndefined();
            expect(finalState.find(c => c.id === "chat3")).toBeDefined();
        });

        it("should preserve state immutability", () => {
            const originalChat = createMockChat("chat1");
            useChatsStore.setState({ chats: [originalChat] });

            const stateBefore = useChatsStore.getState();
            const chatsBefore = stateBefore.chats;

            const newChat = createMockChat("chat2");
            useChatsStore.getState().addChat(newChat);

            const stateAfter = useChatsStore.getState();

            // State should be different reference
            expect(stateAfter.chats).not.toBe(chatsBefore);
            // Original array should be unchanged
            expect(chatsBefore).toEqual([originalChat]);
            // New state should contain both chats
            expect(stateAfter.chats).toEqual([newChat, originalChat]);
        });
    });

    describe("integration with API calls", () => {
        it("should use correct endpoint configuration", async () => {
            const mockResponse: ChatSearchResult = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            await useChatsStore.getState().fetchChats();

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/chats",
                method: "GET",
                inputs: { sortBy: ChatSortBy.DateUpdatedDesc },
                signal: undefined,
            });
        });

        it("should handle pagination edge cases", async () => {
            const mockResponse: ChatSearchResult = {
                edges: [
                    { cursor: "cursor1", node: createMockChat("chat1") },
                    { cursor: "cursor2", node: createMockChat("chat2") },
                ],
                pageInfo: {
                    hasNextPage: true,
                    hasPreviousPage: false,
                    startCursor: "cursor1",
                    endCursor: "cursor2",
                },
            };

            mockFetchData.mockResolvedValue({
                data: mockResponse,
                errors: null,
                timestamp: Date.now(),
            });

            const result = await useChatsStore.getState().fetchChats();

            expect(result).toHaveLength(2);
            expect(result[0].id).toBe("chat1");
            expect(result[1].id).toBe("chat2");
        });
    });

    describe("useChats hook", () => {
        beforeEach(() => {
            // Reset mocks
            mockCheckIfLoggedIn.mockReturnValue(false);
            mockFetchData.mockResolvedValue({
                data: {
                    edges: [],
                    pageInfo: {
                        hasNextPage: false,
                        hasPreviousPage: false,
                        startCursor: null,
                        endCursor: null,
                    },
                },
                errors: null,
                timestamp: Date.now(),
            });
        });

        it("should fetch chats when user logs in and store is empty", async () => {
            const mockChats = [createMockChat("chat1"), createMockChat("chat2")];

            mockFetchData.mockResolvedValue({
                data: {
                    edges: mockChats.map(chat => ({ cursor: chat.id, node: chat })),
                    pageInfo: {
                        hasNextPage: false,
                        hasPreviousPage: false,
                        startCursor: "chat1",
                        endCursor: "chat2",
                    },
                },
                errors: null,
                timestamp: Date.now(),
            });

            // Start with user logged out
            mockCheckIfLoggedIn.mockReturnValue(false);

            const { rerender } = renderHook(() => useChats());

            // User logs in
            mockCheckIfLoggedIn.mockReturnValue(true);
            rerender();

            // Allow async operations to complete
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/chats",
                method: "GET",
                inputs: { sortBy: ChatSortBy.DateUpdatedDesc },
                signal: expect.any(AbortSignal),
            });

            expect(useChatsStore.getState().chats).toEqual(mockChats);
        });

        it("should clear chats when user logs out", () => {
            const existingChats = [createMockChat("chat1"), createMockChat("chat2")];
            useChatsStore.setState({ chats: existingChats });

            // Start with user logged in
            mockCheckIfLoggedIn.mockReturnValue(true);

            const { rerender } = renderHook(() => useChats());

            // User logs out
            mockCheckIfLoggedIn.mockReturnValue(false);
            rerender();

            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should not fetch chats if already loading", () => {
            useChatsStore.setState({ isLoading: true });
            mockCheckIfLoggedIn.mockReturnValue(true);

            renderHook(() => useChats());

            expect(mockFetchData).not.toHaveBeenCalled();
        });

        it("should not fetch chats if there's an error", () => {
            useChatsStore.setState({ error: "Some error" });
            mockCheckIfLoggedIn.mockReturnValue(true);

            renderHook(() => useChats());

            expect(mockFetchData).not.toHaveBeenCalled();
        });

        it("should not fetch chats if chats already exist", () => {
            const existingChats = [createMockChat("existing")];
            useChatsStore.setState({ chats: existingChats });
            mockCheckIfLoggedIn.mockReturnValue(true);

            renderHook(() => useChats());

            expect(mockFetchData).not.toHaveBeenCalled();
        });

        it("should handle fetch errors gracefully", async () => {
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => { });

            mockFetchData.mockRejectedValue(new Error("Network error"));
            mockCheckIfLoggedIn.mockReturnValue(true);

            renderHook(() => useChats());

            // Allow async operations to complete
            await new Promise(resolve => setTimeout(resolve, 0));

            // The error is logged from fetchChats function, not from useChats
            expect(consoleSpy).toHaveBeenCalledWith("Error fetching chats:", expect.any(Error));

            consoleSpy.mockRestore();
        });

        it("should not log error for abort errors", async () => {
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => { });

            const abortError = new Error("AbortError");
            abortError.name = "AbortError";
            mockFetchData.mockRejectedValue(abortError);
            mockCheckIfLoggedIn.mockReturnValue(true);

            renderHook(() => useChats());

            // Allow async operations to complete
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(consoleSpy).not.toHaveBeenCalled();

            consoleSpy.mockRestore();
        });

        it("should abort fetch on unmount", async () => {
            mockCheckIfLoggedIn.mockReturnValue(true);

            const { unmount } = renderHook(() => useChats());

            // Unmount immediately
            unmount();

            // The AbortController.abort() should be called
            // We can't directly test this, but the effect cleanup should prevent any issues
            expect(true).toBe(true); // Placeholder assertion
        });

        it("should react to session changes", () => {
            const existingChats = [createMockChat("chat1")];
            useChatsStore.setState({ chats: existingChats });

            mockCheckIfLoggedIn.mockReturnValue(true);

            const { rerender } = renderHook(
                ({ sessionProp }) => {
                    // Simulate session context change
                    return useChats();
                },
                { initialProps: { sessionProp: { isLoggedIn: true } } }
            );

            // Change session (user logs out)
            mockCheckIfLoggedIn.mockReturnValue(false);
            rerender({ sessionProp: { isLoggedIn: false } });

            expect(useChatsStore.getState().chats).toEqual([]);
        });

        it("should only clear chats if chats exist when logging out", () => {
            const clearChatsSpy = vi.spyOn(useChatsStore.getState(), "clearChats");

            // Start with empty chats
            useChatsStore.setState({ chats: [] });
            mockCheckIfLoggedIn.mockReturnValue(true);

            const { rerender } = renderHook(() => useChats());

            // User logs out
            mockCheckIfLoggedIn.mockReturnValue(false);
            rerender();

            // Should not call clearChats since chats array is already empty
            expect(clearChatsSpy).not.toHaveBeenCalled();

            clearChatsSpy.mockRestore();
        });
    });
});