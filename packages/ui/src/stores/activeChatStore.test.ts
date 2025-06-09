import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useActiveChatStore } from "./activeChatStore.js";

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

    it("setActiveChat should track activeChatId, store chat and participants", () => {
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
        // Call setActiveChat without a session to skip cookie logic
        useActiveChatStore.getState().setActiveChat(dummyChat as any, undefined as any);
        const state = useActiveChatStore.getState();
        expect(state.activeChatId).toBe("chat1");
        expect(state.chats["chat1"]).toEqual(dummyChat);
        expect(state.participants).toEqual(dummyChat.participants);
    });

    it("setLatestMessageId should update latestMessageId", () => {
        const stateBefore = useActiveChatStore.getState();
        expect(stateBefore.latestMessageId).toBeNull();
        useActiveChatStore.getState().setLatestMessageId("msg123");
        const stateAfter = useActiveChatStore.getState();
        expect(stateAfter.latestMessageId).toBe("msg123");
    });

    it("setParticipants and setUsersTyping should update respective arrays", () => {
        const participants = [{ id: "p2" /* omit user */ } as any];
        useActiveChatStore.getState().setParticipants(participants);
        expect(useActiveChatStore.getState().participants).toEqual(participants);

        const typing = [{ id: "p3" /* omit user */ } as any];
        useActiveChatStore.getState().setUsersTyping(typing);
        expect(useActiveChatStore.getState().usersTyping).toEqual(typing);
    });
}); 
