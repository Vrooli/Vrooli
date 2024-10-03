import { combineQueries } from "./combineQueries";

describe("combineQueries", () => {
    describe("basic queries", () => {
        test("handles empty queries", () => {
            const queries = [];
            const expected = {};
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("single query", () => {
            const queries = [{ isActive: true }];
            const expected = { isActive: true };
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines non-conflicting queries", () => {
            const queries = [
                { isDeleted: false },
                { isPrivate: true },
            ];
            const expected = {
                isDeleted: false,
                isPrivate: true,
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines conflicting boolean conditions under AND", () => {
            const queries = [
                { isPrivate: true },
                { isPrivate: false },
            ];
            const expected = {
                AND: [
                    { isPrivate: true },
                    { isPrivate: false },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("merges AND clauses", () => {
            const queries = [
                { AND: [{ isActive: true }] },
                { AND: [{ isVerified: true }] },
            ];
            const expected = {
                AND: [
                    { isActive: true },
                    { isVerified: true },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines conflicting non-boolean conditions under AND", () => {
            const queries = [
                { status: "active" },
                { status: "inactive" },
            ];
            const expected = {
                AND: [
                    { status: "active" },
                    { status: "inactive" },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        describe("handles mixed logical operators", () => {
            describe("mergeMode 'loose'", () => {
                test("with one OR clause", () => {
                    const queries = [
                        { AND: [{ isActive: true }] },
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { NOT: [{ isDeleted: true }] },
                    ];
                    const expected = {
                        AND: [{ isActive: true }],
                        OR: [{ id: 1 }, { id: 2 }],
                        NOT: [{ isDeleted: true }],
                    };
                    expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
                });
                test("with multiple OR clauses", () => {
                    const queries = [
                        { AND: [{ isActive: true }] },
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                        { NOT: [{ isDeleted: true }] },
                    ];
                    const expected = {
                        AND: [{ isActive: true }],
                        OR: [
                            { id: 1 },
                            { id: 2 },
                            { id: 3 },
                        ],
                        NOT: [{ isDeleted: true }],
                    };
                    expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
                });
            });
            describe("mergeMode 'strict'", () => {
                test("with one OR clause", () => {
                    const queries = [
                        { AND: [{ isActive: true }] },
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { NOT: [{ isDeleted: true }] },
                    ];
                    const expected = {
                        AND: [{ isActive: true }],
                        OR: [{ id: 1 }, { id: 2 }],
                        NOT: [{ isDeleted: true }],
                    };
                    expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
                });
                test("with multiple OR clauses", () => {
                    const queries = [
                        { AND: [{ isActive: true }] },
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                        { NOT: [{ isDeleted: true }] },
                    ];
                    const expected = {
                        AND: [
                            { isActive: true },
                            { OR: [{ id: 1 }, { id: 2 }] },
                            { OR: [{ id: 3 }] },
                        ],
                        NOT: [{ isDeleted: true }],
                    };
                    expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
                });
            });
        });

        describe("OR clauses", () => {
            describe("mergeMode 'loose'", () => {
                test("2 OR clauses", () => {
                    const queries = [
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                    ];
                    const expected = {
                        OR: [
                            { id: 1 },
                            { id: 2 },
                            { id: 3 },
                        ],
                    };
                    expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
                });
                test("3 OR clauses", () => {
                    const queries = [
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                        { OR: [{ id: 4 }, { id: 5 }] },
                    ];
                    const expected = {
                        OR: [
                            { id: 1 },
                            { id: 2 },
                            { id: 3 },
                            { id: 4 },
                            { id: 5 },
                        ],
                    };
                    expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
                });
            });
            describe("mergeMode 'strict'", () => {
                test("2 OR clauses", () => {
                    const queries = [
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                    ];
                    const expected = {
                        AND: [
                            { OR: [{ id: 1 }, { id: 2 }] },
                            { OR: [{ id: 3 }] },
                        ],
                    };
                    expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
                });
                test("3 OR clauses", () => {
                    const queries = [
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                        { OR: [{ id: 4 }, { id: 5 }] },
                    ];
                    const expected = {
                        AND: [
                            { OR: [{ id: 1 }, { id: 2 }] },
                            { OR: [{ id: 3 }] },
                            { OR: [{ id: 4 }, { id: 5 }] },
                        ],
                    };
                    expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
                });
            });
        });

        describe("handles conflicting keys with logical operators", () => {
            test("mergeMode 'loose'", () => {
                const queries = [
                    { isPrivate: true },
                    { OR: [{ id: 1 }, { id: 2 }] },
                    { isPrivate: false },
                    { OR: [{ id: 3 }] },
                ];
                const expected = {
                    OR: [
                        { id: 1 },
                        { id: 2 },
                        { id: 3 },
                    ],
                    AND: [
                        { isPrivate: true },
                        { isPrivate: false },
                    ],
                };
                expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
            });
            test("mergeMode 'strict'", () => {
                const queries = [
                    { isPrivate: true },
                    { OR: [{ id: 1 }, { id: 2 }] },
                    { isPrivate: false },
                    { OR: [{ id: 3 }] },
                ];
                const expected = {
                    AND: [
                        { isPrivate: true },
                        { isPrivate: false },
                        { OR: [{ id: 1 }, { id: 2 }] },
                        { OR: [{ id: 3 }] },
                    ],
                };
                const result = combineQueries(queries, { mergeMode: "strict" });
                expect(Object.keys(result)).toEqual(Object.keys(expected));
                expect(result.AND).toEqual(expect.arrayContaining(expected.AND));
                expect(expected.AND).toEqual(expect.arrayContaining(result.AND));
            });
        });

        test("combines queries with conflicting range conditions", () => {
            const queries = [
                { age: { gt: 18 } },
                { age: { lt: 30 } },
                { age: { gt: 25 } },
            ];
            const expected = {
                AND: [
                    {
                        age: {
                            lt: 30, gt: 18, // First occurrences of each key can be combined
                        },
                    },
                    {
                        age: {
                            gt: 25, // Conflicting key cannot be combined
                        },
                    },
                ],
            };
            // NOTE: These are not valid Prisma queries:
            // const expected = {
            //     age: {
            //         AND: [{ gt: 18 }, { lt: 30 }, { gt: 25 }],
            //     },
            // };
            // const expected = {
            //     age: {
            //         gt: 25,
            //         lt: 30,
            //     },
            // };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines queries with NOT operator", () => {
            const queries = [
                {
                    NOT: {
                        email: {
                            endsWith: "hotmail.com",
                        },
                    },
                },
                {
                    NOT: {
                        email: {
                            endsWith: "yahoo.com",
                        },
                    },
                },
            ];
            const expected = {
                NOT: [
                    {
                        email: {
                            endsWith: "hotmail.com",
                        },
                    },
                    {
                        email: {
                            endsWith: "yahoo.com",
                        },
                    },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines queries with 'not' field condition", () => {
            const queries = [
                { content: { not: null } },
                { content: { not: "Test" } },
            ];
            const expected = {
                AND: [
                    { content: { not: null } },
                    { content: { not: "Test" } },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });
    });

    describe("nested queries", () => {
        test("combines nested non-conflicting queries", () => {
            const queries = [
                { user: { id: 1 } },
                { user: { isActive: true } },
            ];
            const expected = {
                user: {
                    id: 1,
                    isActive: true,
                },
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines conflicting complex nested queries under AND", () => {
            const queries = [
                { user: { id: 1, role: "admin" } },
                { user: { id: 2, role: "user" } },
            ];
            const expected = {
                user: {
                    AND: [
                        { id: 1, role: "admin" },
                        { id: 2, role: "user" },
                    ],
                },
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines complex nested queries without conflicts", () => {
            const queries = [
                { user: { id: 1 } },
                { user: { OR: [{ isActive: true }, { isVerified: true }] } },
            ];
            const expected = {
                user: {
                    id: 1,
                    OR: [
                        { isActive: true },
                        { isVerified: true },
                    ],
                },
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("combines conflicting nested queries under AND", () => {
            const queries = [
                { user: { id: 1 } },
                { user: { id: 2 } },
            ];
            const expected = {
                user: {
                    AND: [
                        { id: 1 },
                        { id: 2 },
                    ],
                },
            };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });
    });

    describe("non-object queries", () => {
        test("handles empty and null queries gracefully - test 1", () => {
            const queries = [
                null,
                undefined,
                {},
            ];
            const expected = {};
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });

        test("handles empty and null queries gracefully - test 2", () => {
            const queries = [null, { isActive: true }, undefined, { isVerified: false }];
            const expected = { isActive: true, isVerified: false };
            // Same in both merge modes
            expect(combineQueries(queries)).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).toEqual(expected);
        });
    });

    // Tests that simulate where we use combineQueries in the codebase
    describe("real-world queries", () => {
        describe("counts.ts", () => {
            test("counting public teams", () => {
                const customWhere = undefined;
                const createdQuery = {
                    created_at: {
                        gte: new Date("2021-01-01T00:00:00.000Z").toISOString(),
                    },
                };
                const updatedQuery = {
                    updated_at: {
                        gte: new Date("2022-02-02T00:00:00.000Z").toISOString(),
                        lte: new Date("2023-03-03T23:59:59.999Z").toISOString(),
                    },
                };
                const visibilityQuery = {
                    isPrivate: false,
                };
                const queries = [customWhere, createdQuery, updatedQuery, visibilityQuery];
                const expected = {
                    created_at: {
                        gte: new Date("2021-01-01T00:00:00.000Z").toISOString(),
                    },
                    updated_at: {
                        gte: new Date("2022-02-02T00:00:00.000Z").toISOString(),
                        lte: new Date("2023-03-03T23:59:59.999Z").toISOString(),
                    },
                    isPrivate: false,
                };
                expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            });
        });
        describe("reads.ts", () => {
            test("searching own private routine versions with search string 'boop'", () => {
                const userId = "user123";
                const additionalQueries = { root: { isInternal: true } };
                const createdQuery = undefined;
                const updatedQuery = undefined;
                const visibilityQuery = {
                    isDeleted: false,
                    OR: [
                        {
                            isPrivate: true,
                            root: {
                                isDeleted: false,
                                isPrivate: true,
                                OR: [
                                    { ownedByTeam: { members: { some: { user: { id: userId } } } } },
                                    { ownedByUser: { id: userId } },
                                ],
                            },
                        },
                        {
                            root: {
                                isPrivate: true,
                                isDeleted: false,
                                OR: [
                                    { ownedByTeam: { members: { some: { user: { id: userId } } } } },
                                    { ownedByUser: { id: userId } },
                                ],
                            },
                        },
                    ],
                };
                const queries = [additionalQueries, createdQuery, updatedQuery, visibilityQuery];
                const expected = {
                    isDeleted: false,
                    root: { isInternal: true },
                    OR: [
                        {
                            isPrivate: true,
                            root: {
                                isDeleted: false,
                                isPrivate: true,
                                OR: [
                                    { ownedByTeam: { members: { some: { user: { id: userId } } } } },
                                    { ownedByUser: { id: userId } },
                                ],
                            },
                        },
                        {
                            root: {
                                isPrivate: true,
                                isDeleted: false,
                                OR: [
                                    { ownedByTeam: { members: { some: { user: { id: userId } } } } },
                                    { ownedByUser: { id: userId } },
                                ],
                            },
                        },
                    ],
                };
                expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            });
        });
        describe("feed.ts", () => {
            test("reminders with focus mode", () => {
                const activeFocusModeId = "focusMode123";
                const userId = "user123";
                const customWhere = {
                    reminderList: {
                        focusMode: {
                            id: activeFocusModeId,
                        },
                    },
                };
                const visibilityQuery = {
                    reminderList: {
                        focusMode: {
                            user: {
                                id: userId,
                            },
                        },
                    },
                };
                const queries = [customWhere, visibilityQuery];
                const expected = {
                    reminderList: {
                        focusMode: {
                            id: activeFocusModeId,
                            user: {
                                id: userId,
                            },
                        },
                    },
                };
                expect(combineQueries(queries, { mergeMode: "strict" })).toEqual(expected);
            });
        });
    });
});
