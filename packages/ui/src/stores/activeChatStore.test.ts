// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useActiveChatStore } from "./activeChatStore.js";

// Mock dependencies that aren't in centralized mocks yet
vi.mock("../utils/localStorage.js", () => ({
    getCookieMatchingChat: vi.fn(),
    setCookieMatchingChat: vi.fn(),
}));

vi.mock("../views/objects/chat/ChatCrud.js", () => ({
    CHAT_DEFAULTS: {},
    chatInitialValues: vi.fn(() => ({})),
    transformChatValues: vi.fn(() => ({})),
    withModifiableMessages: vi.fn((chat) => chat),
    withYourMessages: vi.fn((chat) => chat),
}));

vi.mock("../utils/display/translationTools.js", () => ({
    getUserLanguages: vi.fn(() => ["en"]),
}));

describe("ActiveChatStore", () => {
    beforeEach(() => {
        // Reset store state to defaults
        useActiveChatStore.setState({
            activeChatId: null,
            chats: {},
            participants: [],
            usersTyping: [],
            latestMessageId: null,
            isBotOnlyChat: false,
            isLoading: false,
        }, true);
    });

    it("should have initial default values", () => {
        const state = useActiveChatStore.getState();
        expect(state.activeChatId).toBeNull();
        expect(state.chats).toEqual({});
        expect(state.participants).toEqual([]);
        expect(state.usersTyping).toEqual([]);
        expect(state.latestMessageId).toBeNull();
        expect(state.isBotOnlyChat).toBe(false);
        expect(state.isLoading).toBe(false);
    });

    it("should update store state with setState", () => {
        const dummyChat = {
            id: "chat1",
            openToAnyoneWithInvite: false,
            __typename: "Chat",
            invites: [],
            messages: [],
            participants: [{ id: "p1", user: { __typename: "User", id: "u1", handle: "test", isBot: false, name: "Test", profileImage: "" } }],
            participantsDelete: [],
            team: null,
            translations: [],
        };

        // Test direct state updates using setState
        useActiveChatStore.setState({
            activeChatId: "chat1",
            chats: { "chat1": dummyChat as any },
            participants: dummyChat.participants as any,
        });

        const state = useActiveChatStore.getState();
        expect(state.activeChatId).toBe("chat1");
        expect(state.chats["chat1"]).toEqual(dummyChat);
        expect(state.participants).toEqual(dummyChat.participants);
    });

    it("should update latestMessageId with setState", () => {
        const stateBefore = useActiveChatStore.getState();
        expect(stateBefore.latestMessageId).toBeNull();

        useActiveChatStore.setState({ latestMessageId: "msg123" });

        const stateAfter = useActiveChatStore.getState();
        expect(stateAfter.latestMessageId).toBe("msg123");
    });

    it("should update participants and usersTyping with setState", () => {
        const participants = [{ id: "p2" /* omit user */ } as any];
        useActiveChatStore.setState({ participants });
        expect(useActiveChatStore.getState().participants).toEqual(participants);

        const usersTyping = [{ id: "p3" /* omit user */ } as any];
        useActiveChatStore.setState({ usersTyping });
        expect(useActiveChatStore.getState().usersTyping).toEqual(usersTyping);
    });
}); 
