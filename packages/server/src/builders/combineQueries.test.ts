import { expect } from "chai";
import { combineQueries } from "./combineQueries.js";

describe("combineQueries", () => {
    describe("basic queries", () => {
        it("handles empty queries", () => {
            const queries = [];
            const expected = {};
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("single query", () => {
            const queries = [{ isActive: true }];
            const expected = { isActive: true };
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines non-conflicting queries", () => {
            const queries = [
                { isDeleted: false },
                { isPrivate: true },
            ];
            const expected = {
                isDeleted: false,
                isPrivate: true,
            };
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines conflicting boolean conditions under AND", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("merges AND clauses", () => {
            const queries = [
                { AND: [{ isActive: true }] },
                { AND: [{ verifiedAt: true }] },
            ];
            const expected = {
                AND: [
                    { isActive: true },
                    { verifiedAt: true },
                ],
            };
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines conflicting non-boolean conditions under AND", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        describe("handles mixed logical operators", () => {
            describe("mergeMode 'loose'", () => {
                it("with one OR clause", () => {
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
                    expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
                });
                it("with multiple OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
                });
            });
            describe("mergeMode 'strict'", () => {
                it("with one OR clause", () => {
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
                    expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
                });
                it("with multiple OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
                });
            });
        });

        describe("OR clauses", () => {
            describe("mergeMode 'loose'", () => {
                it("2 OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
                });
                it("3 OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
                });
            });
            describe("mergeMode 'strict'", () => {
                it("2 OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
                });
                it("3 OR clauses", () => {
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
                    expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
                });
            });
        });

        describe("handles conflicting keys with logical operators", () => {
            it("mergeMode 'loose'", () => {
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
                expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
            });
            it("mergeMode 'strict'", () => {
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
                expect(Object.keys(result)).to.deep.equal(Object.keys(expected));
                expect(result.AND).to.have.deep.members(expected.AND);
                expect(expected.AND).to.have.deep.members(result.AND);
            });
        });

        it("combines queries with conflicting range conditions", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines queries with NOT operator", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines queries with 'not' field condition", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });
    });

    describe("nested queries", () => {
        it("combines nested non-conflicting queries", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines conflicting complex nested queries under AND", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines complex nested queries without conflicts", () => {
            const queries = [
                { user: { id: 1 } },
                { user: { OR: [{ isActive: true }, { verifiedAt: true }] } },
            ];
            const expected = {
                user: {
                    id: 1,
                    OR: [
                        { isActive: true },
                        { verifiedAt: true },
                    ],
                },
            };
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("combines conflicting nested queries under AND", () => {
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
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });
    });

    describe("non-object queries", () => {
        it("handles empty and null queries gracefully - test 1", () => {
            const queries = [
                null,
                undefined,
                {},
            ];
            const expected = {};
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });

        it("handles empty and null queries gracefully - test 2", () => {
            const queries = [null, { isActive: true }, undefined, { verifiedAt: false }];
            const expected = { isActive: true, verifiedAt: false };
            // Same in both merge modes
            expect(combineQueries(queries)).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            expect(combineQueries(queries, { mergeMode: "loose" })).to.deep.equal(expected);
        });
    });

    // Tests that simulate where we use combineQueries in the codebase
    describe("real-world queries", () => {
        describe("counts.ts", () => {
            it("counting public teams", () => {
                const customWhere = undefined;
                const createdQuery = {
                    createdAt: {
                        gte: new Date("2021-01-01T00:00:00.000Z").toISOString(),
                    },
                };
                const updatedQuery = {
                    updatedAt: {
                        gte: new Date("2022-02-02T00:00:00.000Z").toISOString(),
                        lte: new Date("2023-03-03T23:59:59.999Z").toISOString(),
                    },
                };
                const visibilityQuery = {
                    isPrivate: false,
                };
                const queries = [customWhere, createdQuery, updatedQuery, visibilityQuery];
                const expected = {
                    createdAt: {
                        gte: new Date("2021-01-01T00:00:00.000Z").toISOString(),
                    },
                    updatedAt: {
                        gte: new Date("2022-02-02T00:00:00.000Z").toISOString(),
                        lte: new Date("2023-03-03T23:59:59.999Z").toISOString(),
                    },
                    isPrivate: false,
                };
                expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            });
        });
        describe("reads.ts", () => {
            it("searching own private routine versions with search string 'boop'", () => {
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
                expect(combineQueries(queries, { mergeMode: "strict" })).to.deep.equal(expected);
            });
        });
    });
});
