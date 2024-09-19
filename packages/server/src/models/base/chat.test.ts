import { ChatMessageCreateResult, ChatMessageOperations, ChatPreBranchInfo, PrismaChatMutationResult, prepareChatMessageOperations } from "./chat";

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
                expect(result).toBeUndefined();
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
                expect(result).toBeUndefined();
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
                expect(result).toBeUndefined();
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

                const expected: PrismaChatMutationResult = {
                    create: {
                        id: "1",
                        content: "Hello",
                    },
                };
                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result).toEqual(expected);
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

                const expected: PrismaChatMutationResult = {
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
                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result).toEqual(expected);
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

                const expected: PrismaChatMutationResult = {
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
                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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
                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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
                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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
                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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
                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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
                    expect(() => collectIdsInNestedCreate(result)).not.toThrow();
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
                    expect(() => collectIdsInNestedCreate(result)).not.toThrow();
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

                const expected: PrismaChatMutationResult = {
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

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result).toEqual(expected);
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

                const expected: PrismaChatMutationResult = {
                    delete: [{ id: "A" }],
                    update: [
                        {
                            id: "B",
                            parent: { disconnect: true },
                        },
                    ],
                };

                const result = prepareChatMessageOperations(operations, branchInfo);
                expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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

                    const expected: PrismaChatMutationResult = {
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

                    const result = prepareChatMessageOperations(operations, branchInfo);
                    expect(result).toEqual(expected);
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

            const expected: PrismaChatMutationResult = {
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

            const result = prepareChatMessageOperations(operations, branchInfo);
            expect(result).toEqual(expected);
        });
    });
});
