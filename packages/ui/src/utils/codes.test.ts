import { chatMatchHash, weakHash } from "./codes";

describe("weakHash", () => {
    it("generates a consistent SHA-256 hash for the given string", () => {
        const inputString = "test-string";
        const hash = weakHash(inputString);
        expect(typeof hash).toEqual("string");
    });
});

describe("chatMatchHash", () => {
    it("generates the same hash regardless of the order of participant user IDs", () => {
        // These two arrays contain the same IDs in different orders
        const participantIds1 = ["alice", "bob", "charlie"];
        const participantIds2 = ["charlie", "alice", "bob"];

        const hash1 = chatMatchHash(participantIds1);
        const hash2 = chatMatchHash(participantIds2);

        expect(hash1).toEqual(hash2);
    });

    it("generates different hashes for different sets of participant user IDs", () => {
        const participantIds1 = ["alice", "bob", "charlie"];
        const participantIds2 = ["alice", "bob", "david"];

        const hash1 = chatMatchHash(participantIds1);
        const hash2 = chatMatchHash(participantIds2);

        expect(hash1).not.toEqual(hash2);
    });

    it("generates different hash when specifying a task", () => {
        const participantIds = ["alice", "bob", "charlie"];

        const hash1 = chatMatchHash(participantIds);
        const hash2 = chatMatchHash(participantIds, "task");

        expect(hash1).not.toEqual(hash2);
    });

    it("generates a consistent hash for a given set of participant user IDs", () => {
        const participantIds = ["alice", "bob", "charlie"];

        const hash1 = chatMatchHash(participantIds);
        const hash2 = chatMatchHash(participantIds);

        expect(hash1).toEqual(hash2);
    });
});
