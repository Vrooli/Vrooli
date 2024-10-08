import { ChatMessageCreateResult, ChatMessageOperations, ChatMessageOperationsResult, ChatMessageOperationsSummaryResult, ChatPreBranchInfo, buildChatParticipantMessageQuery, populateMessageDataMap, prepareChatMessageOperations } from "./chat";

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
            chat: { id: "chat1" },
            id: "msg1",
            parent: null,
            translations: [{ id: "123", language: "en", text: "Hello" }],
            user: { id: "user2" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg1"]).toEqual({
            __type: "Update",
            chatId: "chat1",
            messageId: "msg1",
            parentId: undefined,
            translations: [{ id: "123", language: "en", text: "Hello" }],
            userId: "user2",
        });
    });

    test("handles missing translations", () => {
        messages = [{
            chat: { id: "chat2" },
            id: "msg2",
            translations: [],
            user: { id: "user3" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg2"]).toBeUndefined();
    });

    test("recursively populates parent message data", () => {
        messages = [{
            chat: { id: "chat3" },
            id: "msg3",
            parent: {
                chat: { id: "chat2" },
                id: "msg2",
                translations: [{ language: "en", text: "Parent Message" }],
                user: { id: "user3" },
            },
            translations: [{ language: "en", text: "Child Message" }],
            user: { id: "user4" },
        }];

        populateMessageDataMap(messageMap, messages, userData);

        expect(messageMap["msg3"]).toBeDefined();
        expect(messageMap["msg2"]).toBeDefined();
        expect(messageMap["msg2"].messageId).toBe("msg2");
    });

    test("avoids infinite loops with circular references", () => {
        messages = [{
            chat: { id: "chat4" },
            id: "msg4",
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

/**
 * Helper function to ensure that `prepareChatMessageOperations` does not create a cycle in the create operations result
 */
function collectIdsInNestedCreate(
    createResult: ChatMessageCreateResult | undefined,
    collectedIds: Set<string> = new Set(),
): void {
    if (!createResult) return;
    if (collectedIds.has(createResult.id)) {
        throw new Error(`Cycle detected in createResult: ID ${createResult.id} appears more than once`);
    }
    collectedIds.add(createResult.id);

    if (createResult.parent && "create" in createResult.parent) {
        collectIdsInNestedCreate(createResult.parent.create, collectedIds);
    } else if (createResult.parent && "connect" in createResult.parent) {
        // If parent is a connect, we can skip since it's an existing message
    }
}


describe("prepareChatMessageOperations", () => {
    describe("Create Operations", () => {
        describe("Basic Cases", () => {
            it("should handle undefined create operations", () => {
                const operations: ChatMessageOperations = {};
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toBeUndefined();
                expect(result.summary.Create).toHaveLength(0);
                expect(result.summary.Update).toHaveLength(0);
            });

            it("should handle null create operations", () => {
                const operations: ChatMessageOperations = {
                    create: null,
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toBeUndefined();
                expect(result.summary.Create).toHaveLength(0);
                expect(result.summary.Update).toHaveLength(0);
            });

            it("should handle empty array of create operations", () => {
                const operations: ChatMessageOperations = {
                    create: [],
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toBeUndefined();
                expect(result.summary.Create).toHaveLength(0);
                expect(result.summary.Update).toHaveLength(0);
            });

            it("should handle single create operation with no parent connect and no lastSequentialId", () => {
                const operations: ChatMessageOperations = {
                    create: [{
                        id: "1",
                        content: "Hello",
                    }],
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                const expectedOperations: ChatMessageOperationsResult = {
                    create: {
                        id: "1",
                        content: "Hello",
                    },
                };
                const expectedSummary: ChatMessageOperationsSummaryResult = {
                    Create: [{ id: "1", parentId: null }],
                    Update: [],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toEqual(expectedOperations);
                expect(result.summary).toEqual(expectedSummary);
            });

            it("should handle single create operation with no parent connect and a lastSequentialId", () => {
                const operations: ChatMessageOperations = {
                    create: [{
                        id: "1",
                        content: "Hello",
                    }],
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: "123",
                    messageTreeInfo: {},
                };

                const expectedOperations: ChatMessageOperationsResult = {
                    create: {
                        id: "1",
                        content: "Hello",
                        parent: {
                            connect: {
                                id: "123",
                            },
                        },
                    },
                };
                const expectedSummary: ChatMessageOperationsSummaryResult = {
                    Create: [{ id: "1", parentId: "123" }],
                    Update: [],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toEqual(expectedOperations);
                expect(result.summary).toEqual(expectedSummary);
            });

            it("should handle single create operation with a parent connect", () => {
                const operations: ChatMessageOperations = {
                    create: [{
                        id: "1",
                        content: "Hello",
                        parent: {
                            connect: {
                                id: "abc",
                            },
                        },
                    }],
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                const expectedOperations: ChatMessageOperationsResult = {
                    create: {
                        id: "1",
                        content: "Hello",
                        parent: {
                            connect: {
                                id: "abc",
                            },
                        },
                    },
                };
                const expectedSummary: ChatMessageOperationsSummaryResult = {
                    Create: [{ id: "1", parentId: "abc" }],
                    Update: [],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toEqual(expectedOperations);
                expect(result.summary).toEqual(expectedSummary);
            });
        });

        describe("Multiple Creates", () => {
            describe("No parents specified", () => {
                it("sequential ID not specified", () => {
                    const operations: ChatMessageOperations = {
                        create: [
                            { id: "1", content: "Hello" },
                            { id: "2", content: "How are you?" },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: null,
                        messageTreeInfo: {},
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        create: {
                            id: "2",
                            content: "How are you?",
                            parent: {
                                create: {
                                    id: "1",
                                    content: "Hello",
                                },
                            },
                        },
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [
                            { id: "1", parentId: null },
                            { id: "2", parentId: "1" },
                        ],
                        Update: [],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });

                it("sequential ID specified", () => {
                    const operations: ChatMessageOperations = {
                        create: [
                            { id: "1", content: "Hello" },
                            { id: "2", content: "How are you?" },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: "123",
                        messageTreeInfo: {},
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        create: {
                            id: "2",
                            content: "How are you?",
                            parent: {
                                create: {
                                    id: "1",
                                    content: "Hello",
                                    parent: {
                                        connect: {
                                            id: "123",
                                        },
                                    },
                                },
                            },
                        },
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [
                            { id: "1", parentId: "123" },
                            { id: "2", parentId: "1" },
                        ],
                        Update: [],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });
            });

            describe("All parents specified", () => {
                it("sequential ID specified", () => {
                    const operations: ChatMessageOperations = {
                        // Order (leaf to root): 2 -> 1 -> 3 -> 4
                        create: [
                            { id: "1", content: "First message", parent: { connect: { id: "3" } } },
                            { id: "2", content: "Second message", parent: { connect: { id: "1" } } },
                            { id: "3", content: "Third message", parent: { connect: { id: "4" } } },
                            { id: "4", content: "Fourth message" },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: "123",
                        messageTreeInfo: {},
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        create: {
                            id: "2",
                            content: "Second message",
                            parent: {
                                create: {
                                    id: "1",
                                    content: "First message",
                                    parent: {
                                        create: {
                                            id: "3",
                                            content: "Third message",
                                            parent: {
                                                create: {
                                                    id: "4",
                                                    content: "Fourth message",
                                                    parent: {
                                                        connect: {
                                                            id: "123",
                                                        },
                                                    },
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [
                            { id: "4", parentId: "123" },
                            { id: "3", parentId: "4" },
                            { id: "1", parentId: "3" },
                            { id: "2", parentId: "1" },
                        ],
                        Update: [],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });
            });

            describe("Some parents specified", () => {
                it("sequential ID specified", () => {
                    const operations: ChatMessageOperations = {
                        // Order (leaf to root): 1 -> 3 -> 4 -> 2
                        create: [
                            { id: "1", content: "First message", parent: { connect: { id: "3" } } },
                            { id: "2", content: "Second message" },
                            { id: "3", content: "Third message", parent: { connect: { id: "4" } } },
                            { id: "4", content: "Fourth message" },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: "123",
                        messageTreeInfo: {},
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        create: {
                            id: "1",
                            content: "First message",
                            parent: {
                                create: {
                                    id: "3",
                                    content: "Third message",
                                    parent: {
                                        create: {
                                            id: "4",
                                            content: "Fourth message",
                                            parent: {
                                                create: {
                                                    id: "2",
                                                    content: "Second message",
                                                    parent: {
                                                        connect: {
                                                            id: "123",
                                                        },
                                                    },
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [
                            { id: "2", parentId: "123" },
                            { id: "4", parentId: "2" },
                            { id: "3", parentId: "4" },
                            { id: "1", parentId: "3" },
                        ],
                        Update: [],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });
            });
        });

        describe("Error Handling", () => {
            describe("should handle cycles", () => {
                it("one big cycle", () => {
                    const operations: ChatMessageOperations = {
                        create: [
                            { id: "1", content: "First message", parent: { connect: { id: "3" } } },
                            { id: "2", content: "Second message", parent: { connect: { id: "1" } } },
                            { id: "3", content: "Third message", parent: { connect: { id: "2" } } },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: "123",
                        messageTreeInfo: {},
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(() => collectIdsInNestedCreate(result.operations?.create)).not.toThrow();
                });

                it("two small cycles", () => {
                    const operations: ChatMessageOperations = {
                        create: [
                            { id: "1", content: "First message", parent: { connect: { id: "2" } } },
                            { id: "2", content: "Second message", parent: { connect: { id: "1" } } },
                            { id: "3", content: "Third message", parent: { connect: { id: "4" } } },
                            { id: "4", content: "Fourth message", parent: { connect: { id: "3" } } },
                        ],
                    };
                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: "123",
                        messageTreeInfo: {},
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(() => collectIdsInNestedCreate(result.operations?.create)).not.toThrow();
                });
            });

            it("should throw an error branching is detected", () => {
                const operations: ChatMessageOperations = {
                    create: [
                        { id: "1", content: "First message", parent: { connect: { id: "3" } } },
                        { id: "2", content: "Second message", parent: { connect: { id: "3" } } },
                        { id: "3", content: "Third message" },
                    ],
                };
                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: "123",
                    messageTreeInfo: {},
                };

                expect(() => prepareChatMessageOperations(operations, branchInfo)).toThrow();
            });
        });
    });

    describe("Delete Operations", () => {
        describe("Reparenting Children", () => {
            it("should re-parent children to the nearest non-deleted ancestor when a message is deleted", () => {
                const operations: ChatMessageOperations = {
                    delete: [{ id: "B" }],
                };

                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: "D",
                    messageTreeInfo: {
                        "B": { parentId: "A", childIds: ["C", "D"] },
                    },
                };

                const expectedOperations: ChatMessageOperationsResult = {
                    delete: [{ id: "B" }],
                    update: [
                        {
                            id: "C",
                            parent: { connect: { id: "A" } },
                        },
                        {
                            id: "D",
                            parent: { connect: { id: "A" } },
                        },
                    ],
                };
                const expectedSummary: ChatMessageOperationsSummaryResult = {
                    Create: [],
                    Update: [
                        { id: "C", parentId: "A" },
                        { id: "D", parentId: "A" },
                    ],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toEqual(expectedOperations);
                expect(result.summary).toEqual(expectedSummary);
            });

            it("should disconnect children if the deleted message has no parent", () => {
                const operations: ChatMessageOperations = {
                    delete: [{ id: "A" }],
                };

                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {
                        "A": { parentId: null, childIds: ["B"] },
                    },
                };

                const expectedOperations: ChatMessageOperationsResult = {
                    delete: [{ id: "A" }],
                    update: [
                        {
                            id: "B",
                            parent: { disconnect: true },
                        },
                    ],
                };
                const expectedSummary: ChatMessageOperationsSummaryResult = {
                    Create: [],
                    Update: [
                        { id: "B", parentId: null },
                    ],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result.operations).toEqual(expectedOperations);
                expect(result.summary).toEqual(expectedSummary);
            });

            describe("should handle multiple messages being deleted simultaneously", () => {
                it("deleted message's parent is also deleted", () => {
                    const operations: ChatMessageOperations = {
                        delete: [{ id: "C" }, { id: "D" }],
                    };

                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: null,
                        messageTreeInfo: {
                            "C": { parentId: "B", childIds: ["D"] },
                            "D": { parentId: "C", childIds: ["E", "F"] },
                        },
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        delete: [{ id: "C" }, { id: "D" }],
                        update: [
                            {
                                id: "E",
                                parent: { connect: { id: "B" } },
                            },
                            {
                                id: "F",
                                parent: { connect: { id: "B" } },
                            },
                        ],
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [],
                        Update: [
                            { id: "E", parentId: "B" },
                            { id: "F", parentId: "B" },
                        ],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });

                it("deleted message's child is also deleted", () => {
                    const operations: ChatMessageOperations = {
                        delete: [{ id: "C" }, { id: "D" }],
                    };

                    const branchInfo: ChatPreBranchInfo = {
                        lastSequenceId: null,
                        messageTreeInfo: {
                            "C": { parentId: "B", childIds: ["D", "E"] },
                            "D": { parentId: "A", childIds: ["F"] },
                        },
                    };

                    const expectedOperations: ChatMessageOperationsResult = {
                        delete: [{ id: "C" }, { id: "D" }],
                        update: [
                            {
                                id: "E",
                                parent: { connect: { id: "B" } },
                            },
                            {
                                id: "F",
                                parent: { connect: { id: "A" } },
                            },
                        ],
                    };
                    const expectedSummary: ChatMessageOperationsSummaryResult = {
                        Create: [],
                        Update: [
                            { id: "E", parentId: "B" },
                            { id: "F", parentId: "A" },
                        ],
                    };

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result.operations).toEqual(expectedOperations);
                    expect(result.summary).toEqual(expectedSummary);
                });
            });

            it("should throw an error if messageTreeInfo is missing for a deleted message", () => {
                const operations: ChatMessageOperations = {
                    delete: [{ id: "A" }],
                };

                const branchInfo: ChatPreBranchInfo = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };

                expect(() => prepareChatMessageOperations(operations, branchInfo)).toThrow();
            });
        });
    });

    describe("Combined Create, Update, and Delete Operations", () => {
        it("should correctly process create, update, and delete operations together", () => {
            const operations: ChatMessageOperations = {
                create: [
                    { id: "3", content: "First message", parent: { connect: { id: "4" } } },
                    { id: "4", content: "Second message", parent: { connect: { id: "2" } } },
                ],
                update: [
                    { id: "1", content: "Third message" },
                    { id: "2", content: "Fourth message" },
                ],
                delete: [{ id: "5" }],
            };

            const branchInfo: ChatPreBranchInfo = {
                lastSequenceId: "123",
                messageTreeInfo: {
                    "5": { parentId: "1", childIds: ["2", "6"] },
                },
            };

            const expectedOperations: ChatMessageOperationsResult = {
                create: {
                    id: "3",
                    content: "First message",
                    parent: {
                        create: {
                            id: "4",
                            content: "Second message",
                            parent: {
                                connect: {
                                    id: "2",
                                },
                            },
                        },
                    },
                },
                update: [
                    {
                        id: "1",
                        content: "Third message",
                    },
                    {
                        id: "2",
                        content: "Fourth message",
                        parent: { connect: { id: "1" } },
                    },
                    {
                        id: "6",
                        parent: { connect: { id: "1" } },
                    },
                ],
                delete: [{ id: "5" }],
            };
            const expectedSummary: ChatMessageOperationsSummaryResult = {
                Create: [
                    { id: "4", parentId: "2" },
                    { id: "3", parentId: "4" },
                ],
                Update: [
                    // Note that "1" isn't included here because it's not being re-parented. VERY IMPORTANT
                    { id: "2", parentId: "1" },
                    { id: "6", parentId: "1" },
                ],
            };

            const result = prepareChatMessageOperations(operations, branchInfo);
            expect(result.operations).toEqual(expectedOperations);
            expect(result.summary).toEqual(expectedSummary);
        });
    });
});
