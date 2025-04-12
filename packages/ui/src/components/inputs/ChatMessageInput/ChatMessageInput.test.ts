import { expect } from "chai";
import { getTypingIndicatorText } from "./ChatMessageInput.js";

describe("getTypingIndicatorText", () => {

    it("returns an empty string when there are no participants", () => {
        expect(getTypingIndicatorText([], 50)).to.equal("");
    });

    it("handles case where maxChars is too small to include any names with ChatParticipants", () => {
        const participants = [
            { user: { id: "1", name: "Alice", __typename: "User" as const }, __typename: "ChatParticipant" as const },
            { user: { id: "2", name: "Bob", __typename: "User" as const }, __typename: "ChatParticipant" as const },
        ];
        expect(getTypingIndicatorText(participants, 5)).to.equal("+2 are typing");
    });

    it("handles ChatParticipants with names that have special characters", () => {
        const participants = [
            { user: { id: "1", name: "Alice_123!", __typename: "User" as const }, __typename: "ChatParticipant" as const },
            { user: { id: "2", name: "Bob$@#", __typename: "User" as const }, __typename: "ChatParticipant" as const },
            { user: { id: "3", name: "Charlie%^&", __typename: "User" as const }, __typename: "ChatParticipant" as const },
        ];
        expect(getTypingIndicatorText(participants, 50)).to.equal("Alice_123!, Bob$@#, Charlie%^& are typing");
    });

    it("handles a mix of ChatParticipants and direct user objects", () => {
        const participants = [
            { user: { id: "1", name: "Alice", __typename: "User" as const }, __typename: "ChatParticipant" as const },
            { id: "2", name: "Bob", __typename: "User" as const },
        ];
        expect(getTypingIndicatorText(participants, 50)).to.equal("Alice, Bob are typing");
    });

    describe("length checks", () => {
        describe("single participant", () => {
            it("name shorter than maxChars - ' is typing'", () => {
                const participants = [
                    { user: { id: "1", name: "1234567890", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 20;
                const result = getTypingIndicatorText(participants, maxChars);
                expect(result.length).to.be.at.most(maxChars);
                expect(result).to.equal("1234567890 is typing");
            });
            it("name longer than maxChars - ' is typing'", () => {
                const participants = [
                    { user: { id: "1", name: "12345678901234567890", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 23;
                const result = getTypingIndicatorText(participants, maxChars);
                expect(result.length).to.be.at.most(maxChars);
                expect(result).to.equal("123456789012… is typing"); // Cuts off enough characters to add "... is typing"
            });
            it("name longer than maxChars", () => {
                const participants = [
                    { user: { id: "1", name: "123456789012345678901234567890", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 23;
                const result = getTypingIndicatorText(participants, maxChars);
                expect(result.length).to.be.at.most(maxChars);
                expect(result).to.equal("123456789012… is typing");
            });
        });
        describe("multiple participants", () => {
            it("names fit", () => {
                const participants = [
                    { user: { id: "1", name: "Alice", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "2", name: "Bob", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "3", name: "Charlie", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "4", name: "David", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 100;
                const expected = "Alice, Bob, Charlie, David are typing";
                expect(getTypingIndicatorText(participants, maxChars)).to.equal(expected);
                expect(expected.length).to.be.at.most(maxChars);
            });
            it("some names cut off - ellipsis in name", () => {
                const participants = [
                    { user: { id: "1", name: "Alice", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "2", name: "Bob Odenkirk", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "3", name: "Charlie", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "4", name: "David", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 24;
                const expected = "Alice, B…, +2 are typing";
                expect(getTypingIndicatorText(participants, maxChars)).to.equal(expected);
                expect(expected.length).to.equal(maxChars);
            });
            it("some names cut off - ellipsis not in name", () => {
                const participants = [
                    { user: { id: "1", name: "Alice", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "2", name: "Bob", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "3", name: "Charlie", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                    { user: { id: "4", name: "David", __typename: "User" as const }, __typename: "ChatParticipant" as const },
                ];
                const maxChars = 20;
                const expected = "Alice, +3 are typing";
                expect(getTypingIndicatorText(participants, maxChars)).to.equal(expected);
                expect(expected.length).to.equal(maxChars);
            });
        });
    });
});
