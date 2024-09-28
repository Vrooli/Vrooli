import { combineQueries } from "./combineQueries";

describe("combineQueries", () => {
    describe("basic queries", () => {
        test("combines non-conflicting queries", () => {
            const queries = [
                { isDeleted: false },
                { isPrivate: true },
            ];
            const expected = {
                isDeleted: false,
                isPrivate: true,
            };
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
        });

        test("merges OR clauses", () => {
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
        });

        test("handles mixed logical operators", () => {
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
            expect(combineQueries(queries)).toEqual(expected);
        });

        test("merges multiple OR clauses correctly", () => {
            const queries = [
                { OR: [{ id: 1 }] },
                { OR: [{ id: 2 }] },
                { OR: [{ id: 3 }] },
            ];
            const expected = {
                OR: [
                    { id: 1 },
                    { id: 2 },
                    { id: 3 },
                ],
            };
            expect(combineQueries(queries)).toEqual(expected);
        });

        test("handles conflicting keys with logical operators", () => {
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
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
            expect(combineQueries(queries)).toEqual(expected);
        });

        test("handles empty and null queries gracefully - test 2", () => {
            const queries = [null, { isActive: true }, undefined, { isVerified: false }];
            const expected = { isActive: true, isVerified: false };
            expect(combineQueries(queries)).toEqual(expected);
        });
    });

    // describe("real-world queries", () => {

    // });
});
