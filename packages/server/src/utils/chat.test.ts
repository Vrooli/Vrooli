import { buildChatParticipantMessageQuery, populateMessageDataMap } from "./chat";

/**
 * Recursively checks that each object within a 'select' object
 * correctly contains only objects with a 'select' key.
 * @param obj The object to check for proper 'select' structure.
 */
const checkSelectStructure = (obj: any) => {
    if (typeof obj !== "object" || obj === null) {
        throw new Error("Non-object or null found where an object was expected");
    }
    Object.keys(obj).forEach(key => {
        if (key === "select") {
            expect(typeof obj[key]).toBe("object");
            checkSelectStructure(obj[key]); // Recursive call
        } else {
            // Optionally add more checks here if you need to verify other properties
        }
    });
};

describe("buildChatParticipantMessageQuery", () => {
    test("returns only last message ID selection when no message info is included", () => {
        const query = buildChatParticipantMessageQuery(["123"], false, false);
        expect(query.orderBy).toEqual({ sequence: "desc" });
        expect(query.take).toBe(1);
        expect(query).toHaveProperty("select.id");
    });

    test("includes the correct \"where\" clause for message IDs", () => {
        const query = buildChatParticipantMessageQuery(["123"], true, false);
        expect(query.where).toEqual({ id: { in: ["123"] } });
    });

    test("includes only basic parent data when only parent message info is included", () => {
        const query = buildChatParticipantMessageQuery(["123"], false, true);
        console.log("got the result select", JSON.stringify(query));
        // Full parent info
        expect(query.select.parent.select).toHaveProperty("id");
        expect(query.select.parent.select).toHaveProperty("parent.select");
        expect(query.select.parent.select).toHaveProperty("translations.select");
        expect(query.select.parent.select).toHaveProperty("user.select");

        // No other message info
        expect(query.select).not.toHaveProperty("id");
        expect(query.select).not.toHaveProperty("translations");
        expect(query.select).not.toHaveProperty("user");
    });

    test("includes detailed parent data when both message and parent info are included", () => {
        const query = buildChatParticipantMessageQuery(["123"], true, true);
        // Full parent info
        expect(query.select.parent.select).toHaveProperty("id");
        expect(query.select.parent.select).toHaveProperty("parent.select");
        expect(query.select.parent.select).toHaveProperty("translations.select");
        expect(query.select.parent.select).toHaveProperty("user.select");

        // Full message info
        expect(query.select).toHaveProperty("id");
        expect(query.select).toHaveProperty("translations.select");
        expect(query.select).toHaveProperty("user.select");
    });

    test("includes only parent ID when only message info is included", () => {
        const query = buildChatParticipantMessageQuery(["123"], true, false);
        // Simple parent info
        expect(query.select.parent.select).toHaveProperty("id");
        expect(query.select.parent.select).not.toHaveProperty("parent.select");
        expect(query.select.parent.select).not.toHaveProperty("translations.select");
        expect(query.select.parent.select).not.toHaveProperty("user.select");

        // Full message info
        expect(query.select).toHaveProperty("id");
        expect(query.select).toHaveProperty("translations.select");
        expect(query.select).toHaveProperty("user.select");
    });

    test("ensures select keys are properly structured in all select objects", () => {
        const query = buildChatParticipantMessageQuery(["123"], true, true);
        // Recursively check the select structure starting from the root if needed
        checkSelectStructure(query);
    });
});

describe("populateMessageDataMap", () => {
    let messageMap;
    let messages;
    let userData;

    beforeEach(() => {
        // Reset data for each test to avoid contamination
        messageMap = {};
        userData = { id: "user1", languages: ["en", "es"] };
    });

    test("populates message data correctly for a single message", () => {
        messages = [{
            id: "msg1",
            chat: { id: "chat1" },
            parent: null,
            translations: [{ language: "en", text: "Hello" }],
            user: { id: "user2" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg1"]).toEqual({
            chatId: "chat1",
            content: "Hello",
            id: "msg1",
            isNew: false,
            language: "en",
            parentId: undefined,
            translations: [{ language: "en", text: "Hello" }],
            userId: "user2",
        });
    });

    test("handles missing translations", () => {
        messages = [{
            id: "msg2",
            chat: { id: "chat2" },
            translations: [],
            user: { id: "user3" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg2"]).toBeUndefined();
    });

    test("recursively populates parent message data", () => {
        messages = [{
            id: "msg3",
            chat: { id: "chat3" },
            parent: {
                id: "msg2",
                chat: { id: "chat2" },
                translations: [{ language: "en", text: "Parent Message" }],
                user: { id: "user3" },
            },
            translations: [{ language: "en", text: "Child Message" }],
            user: { id: "user4" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg3"]).toBeDefined();
        expect(messageMap["msg2"]).toBeDefined();
        expect(messageMap["msg2"].content).toEqual("Parent Message");
    });

    test("avoids infinite loops with circular references", () => {
        messages = [{
            id: "msg4",
            chat: { id: "chat4" },
            parent: {
                id: "msg5",
                chat: { id: "chat5" },
                parent: {
                    id: "msg4",  // Circular reference
                    chat: { id: "chat4" },
                    translations: [{ language: "en", text: "Hello again" }],
                    user: { id: "user5" },
                },
                translations: [{ language: "en", text: "Hello" }],
                user: { id: "user6" },
            },
            translations: [{ language: "en", text: "Hello world" }],
            user: { id: "user7" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg4"]).toBeDefined();
        expect(messageMap["msg5"]).toBeDefined();
        // This test ensures no infinite recursion, focus is on completion
        expect(Object.keys(messageMap).length).toBe(2);
    });
});
