import { expect } from "chai";
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
        expect(state.activeChatId).to.be.null;
        expect(state.chats).to.deep.equal({});
        expect(state.participants).to.deep.equal([]);
        expect(state.usersTyping).to.deep.equal([]);
        expect(state.latestMessageId).to.be.null;
        expect(state.isBotOnlyChat).to.be.false;
        expect(state.isLoading).to.be.false;
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
        expect(state.activeChatId).to.equal("chat1");
        expect(state.chats["chat1"]).to.deep.equal(dummyChat);
        expect(state.participants).to.deep.equal(dummyChat.participants);
    });

    it("setLatestMessageId should update latestMessageId", () => {
        const stateBefore = useActiveChatStore.getState();
        expect(stateBefore.latestMessageId).to.be.null;
        useActiveChatStore.getState().setLatestMessageId("msg123");
        const stateAfter = useActiveChatStore.getState();
        expect(stateAfter.latestMessageId).to.equal("msg123");
    });

    it("setParticipants and setUsersTyping should update respective arrays", () => {
        const participants = [{ id: "p2" /* omit user */ } as any];
        useActiveChatStore.getState().setParticipants(participants);
        expect(useActiveChatStore.getState().participants).to.deep.equal(participants);

        const typing = [{ id: "p3" /* omit user */ } as any];
        useActiveChatStore.getState().setUsersTyping(typing);
        expect(useActiveChatStore.getState().usersTyping).to.deep.equal(typing);
    });
}); 
