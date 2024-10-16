import { ModelMap } from "../models/base/index";
import { JoinType, SqlBuilder } from "./sqlBuilder";

jest.mock("../models/base/index");

describe("SqlBuilder", () => {
    let builder: SqlBuilder;
    let originalModelMap;

    beforeEach(() => {
        jest.clearAllMocks();
        originalModelMap = { ...ModelMap };
        Object.assign(ModelMap, {
            get: jest.fn().mockImplementation((objectType) => ({
                dbTable: objectType + "_table",
            })),
        });

        builder = new SqlBuilder("Meeting");
    });

    afterEach(() => {
        Object.assign(ModelMap, originalModelMap);
    });

    test("initializes with correct FROM clause", () => {
        expect(builder.serialize()).toContain("FROM \"Meeting_table\" AS \"a\"");
    });

    test("uses '*' when no SELECT fields are added", () => {
        expect(builder.serialize()).toContain("SELECT * FROM");
    });

    test("correctly assigns aliases and builds SELECT clause", () => {
        builder.addSelect("Meeting", "id");
        builder.addSelect("Meeting", "title", "meetingTitle");
        expect(builder.serialize()).toEqual("SELECT \"a\".\"id\", \"a\".\"title\" AS \"meetingTitle\" FROM \"Meeting_table\" AS \"a\";");
    });

    test("adds a raw SQL expression to the SELECT clause", () => {
        const rawSQL = "COUNT(DISTINCT \"user_id\") AS \"uniqueUsers\"";
        builder.addSelectRaw(rawSQL);
        expect(builder.serialize()).toEqual(`SELECT ${rawSQL} FROM "Meeting_table" AS "a";`);
    });

    test("handles multiple raw SQL expressions in the SELECT clause", () => {
        const rawSQL1 = "COUNT(DISTINCT \"user_id\") AS \"uniqueUsers\"";
        const rawSQL2 = "SUM(\"duration\") AS \"totalDuration\"";
        builder.addSelectRaw(rawSQL1);
        builder.addSelectRaw(rawSQL2);
        expect(builder.serialize()).toEqual(`SELECT ${rawSQL1}, ${rawSQL2} FROM "Meeting_table" AS "a";`);
    });

    test("handles both SELECT fields and raw SQL expressions", () => {
        builder.addSelect("Meeting", "id");
        const rawSQL = "COUNT(DISTINCT \"user_id\") AS \"uniqueUsers\"";
        builder.addSelectRaw(rawSQL);
        expect(builder.serialize()).toEqual(`SELECT "a"."id", ${rawSQL} FROM "Meeting_table" AS "a";`);
    });

    test("adds JOIN clauses correctly with automatic alias management", () => {
        builder.addJoin("User", JoinType.LEFT, SqlBuilder.equals(builder.field("Meeting", "userId"), builder.field("User", "id")));
        expect(builder.serialize()).toContain("LEFT JOIN \"User_table\" AS \"b\" ON \"a\".\"userId\" = \"b\".\"id\"");
    });

    test("handles WHERE conditions correctly", () => {
        builder.addWhere(SqlBuilder.equals(builder.field("Meeting", "isActive"), true));
        expect(builder.serialize()).toContain("WHERE \"a\".\"isActive\" = true");
    });

    test("manages multiple joins and aliases without collisions", () => {
        builder.addJoin("User", JoinType.LEFT, SqlBuilder.equals(builder.field("Meeting", "userId"), builder.field("User", "id")));
        builder.addJoin("Team", JoinType.INNER, SqlBuilder.equals(builder.field("Meeting", "orgId"), builder.field("Team", "id")));
        expect(builder.serialize()).toEqual("SELECT * FROM \"Meeting_table\" AS \"a\" LEFT JOIN \"User_table\" AS \"b\" ON \"a\".\"userId\" = \"b\".\"id\" INNER JOIN \"Team_table\" AS \"c\" ON \"a\".\"orgId\" = \"c\".\"id\";");
    });

    describe("addOrderBy function", () => {
        test("sets a single ORDER BY clause correctly", () => {
            builder.addOrderBy("Meeting", "points", "DESC");
            expect(builder.query.orderBy).toEqual(["\"a\".\"points\" DESC"]);
        });

        test("sets multiple ORDER BY clauses correctly", () => {
            builder.addOrderBy("User", "points", "DESC");
            builder.addOrderBy("Meeting", "date", "ASC");
            expect(builder.query.orderBy).toEqual(["\"b\".\"points\" DESC", "\"a\".\"date\" ASC"]);
        });

        test("handles different object types and fields", () => {
            builder.addOrderBy("User", "score", "ASC");
            expect(builder.query.orderBy).toContain("\"b\".\"score\" ASC");
        });
    });

    describe("setLimit function", () => {
        test("sets the limit correctly", () => {
            builder.setLimit(10);
            expect(builder.query.limit).toBe(10);
        });

        test("integrates limit into the full query serialization", () => {
            builder.addSelect("Meeting", "id");
            builder.setLimit(5);
            const expectedQuery = "SELECT \"a\".\"id\" FROM \"Meeting_table\" AS \"a\" LIMIT 5;";
            expect(builder.serialize()).toEqual(expectedQuery);
        });

        test("throws an error if limit is not valid", () => {
            expect(() => builder.setLimit(-5)).toThrow();
            expect(() => builder.setLimit(0)).toThrow();
            expect(() => builder.setLimit("five" as any)).toThrow();
            expect(() => builder.setLimit(5.5)).toThrow();
            expect(() => builder.setLimit(Infinity)).toThrow();
        });
    });

    describe("setOffset function", () => {
        test("sets the offset correctly", () => {
            builder.setOffset(30);
            expect(builder.query.offset).toBe(30);
        });

        test("integrates offset into the full query serialization", () => {
            builder.addSelect("Meeting", "id");
            builder.setLimit(10);  // Set limit to show interaction with offset
            builder.setOffset(20);
            const expectedQuery = "SELECT \"a\".\"id\" FROM \"Meeting_table\" AS \"a\" LIMIT 10 OFFSET 20;";
            expect(builder.serialize()).toEqual(expectedQuery);
        });

        test("throws an error if offset is not valid", () => {
            expect(() => builder.setOffset(-1)).toThrow();
            expect(() => builder.setOffset("twenty" as any)).toThrow();
            expect(() => builder.setOffset(5.5)).toThrow();
            expect(() => builder.setOffset(Infinity)).toThrow();
            expect(() => builder.setOffset(NaN)).toThrow();
        });
    });

    test("throws error when exceeding alias limit", () => {
        expect(() => {
            for (let i = 0; i < 27; i++) {  // Exceeds the 26 character limit
                builder.addJoin(`Table${i}` as any, JoinType.LEFT, `Meeting.id = Table${i}.meetingId`);
            }
        }).toThrow();
    });

    describe("serialize function", () => {
        test("serialize test 1", () => {
            builder.addSelect("Meeting", "id");
            builder.addJoin("User", JoinType.LEFT, SqlBuilder.equals(builder.field("Meeting", "userId"), builder.field("User", "id")));
            builder.addWhere(SqlBuilder.equals(builder.field("Meeting", "isActive"), true));
            builder.addWhere(SqlBuilder.equals(builder.field("User", "isVerified"), true));
            builder.addOrderBy("Meeting", "title", "ASC");
            expect(builder.serialize()).toEqual("SELECT \"a\".\"id\" FROM \"Meeting_table\" AS \"a\" LEFT JOIN \"User_table\" AS \"b\" ON \"a\".\"userId\" = \"b\".\"id\" WHERE \"a\".\"isActive\" = true AND \"b\".\"isVerified\" = true ORDER BY \"a\".\"title\" ASC;");
        });

        test("serialize test 2", () => {
            builder.addSelect("Meeting", "id");
            builder.addSelect("User", "name");
            builder.addJoin("User", JoinType.INNER, SqlBuilder.equals(builder.field("Meeting", "userId"), builder.field("User", "id")));
            builder.addJoin("Post", JoinType.LEFT, SqlBuilder.equals(builder.field("Meeting", "locationId"), builder.field("Post", "id")));
            builder.addWhere(SqlBuilder.and(
                SqlBuilder.equals(builder.field("Meeting", "isActive"), true),
                SqlBuilder.equals(builder.field("Post", "isAvailable"), true),
            ));
            builder.setLimit(10);
            builder.setOffset(20);
            const expectedQuery = "SELECT \"a\".\"id\", \"b\".\"name\" FROM \"Meeting_table\" AS \"a\" INNER JOIN \"User_table\" AS \"b\" ON \"a\".\"userId\" = \"b\".\"id\" LEFT JOIN \"Post_table\" AS \"c\" ON \"a\".\"locationId\" = \"c\".\"id\" WHERE (\"a\".\"isActive\" = true) AND (\"c\".\"isAvailable\" = true) LIMIT 10 OFFSET 20;";
            expect(builder.serialize()).toEqual(expectedQuery);
        });

        test("serialize test 3", () => {
            builder.addSelect("User", "id");
            builder.addSelect("User", "signupDate");
            builder.addOrderByRaw("EXTRACT(YEAR FROM \"b\".\"signupDate\") DESC");
            builder.addOrderBy("User", "name", "ASC");
            // Technically invalid SQL because we never joined with the "User" table, but it's just a test
            const expectedQuery = "SELECT \"b\".\"id\", \"b\".\"signupDate\" FROM \"Meeting_table\" AS \"a\" ORDER BY EXTRACT(YEAR FROM \"b\".\"signupDate\") DESC, \"b\".\"name\" ASC;";
            expect(builder.serialize()).toEqual(expectedQuery);
        });

        test("serialize test 4", () => {
            builder.addSelect("Meeting", "id");
            builder.addWhere(SqlBuilder.or(
                SqlBuilder.equals(builder.field("Meeting", "status"), "confirmed"),
                SqlBuilder.and(
                    SqlBuilder.equals(builder.field("Meeting", "status"), "pending"),
                    SqlBuilder.equals(builder.field("Meeting", "priority"), "high"),
                ),
            ));
            const expectedQuery = "SELECT \"a\".\"id\" FROM \"Meeting_table\" AS \"a\" WHERE (\"a\".\"status\" = 'confirmed') OR ((\"a\".\"status\" = 'pending') AND (\"a\".\"priority\" = 'high'));";
            expect(builder.serialize()).toEqual(expectedQuery);
        });
    });

    describe("field function", () => {
        test("field generates correct SQL field reference object", () => {
            const fieldRef = builder.field("User", "id");
            expect(fieldRef).toEqual({
                tableAlias: "b", // Should be the second alias generated, since 'a' is already taken by the root
                column: "id",
            });
        });
    });

    describe("embedPoints function", () => {
        test("EmbedTopAsc generates correct SQL", () => {
            const objectType = "Meeting";
            const translationObjectType = "MeetingTranslation" as any;
            const searchStringEmbedding = [3, 2];
            builder.embedPoints(translationObjectType, objectType, searchStringEmbedding, "EmbedTopAsc");
            expect(builder.query.select).toContainEqual(
                "(1 / (POWER((\"b\".\"embedding\" <-> ARRAY[3.000000, 2.000000]::vector) + 0.01, 2))) + (EXP(-\"a\".\"bookmarks\" / 5) * 20) AS points",
            );
        });

        test("EmbedTopDesc generates correct SQL", () => {
            const objectType = "User";
            const translationObjectType = "UserTranslation" as any;
            const searchStringEmbedding = [1];
            builder.embedPoints(translationObjectType, objectType, searchStringEmbedding, "EmbedTopDesc");
            expect(builder.query.select).toContainEqual(
                "(1 / (POWER((\"c\".\"embedding\" <-> ARRAY[1.000000]::vector) + 0.01, 2))) + (LN(\"b\".\"bookmarks\" + 1) * 2) AS points",
            );
        });

        test("EmbedDateCreatedAsc generates correct SQL", () => {
            const objectType = "Comment";
            const translationObjectType = "CommentTranslation" as any;
            const searchStringEmbedding = [8.3828191932843];
            builder.embedPoints(translationObjectType, objectType, searchStringEmbedding, "EmbedDateCreatedAsc");
            expect(builder.query.select).toContainEqual(
                "(1 / (POWER((\"c\".\"embedding\" <-> ARRAY[8.382819]::vector) + 0.01, 2))) + (LOG(1 + POWER(ABS(EXTRACT(EPOCH FROM NOW() - \"b\".\"created_at\") / 3600), 0.25)) * 1) AS points",
            );
        });

        test("EmbedDateUpdatedDesc generates correct SQL", () => {
            const objectType = "User";
            const translationObjectType = "UserTranslation" as any;
            const searchStringEmbedding = [];
            builder.embedPoints(translationObjectType, objectType, searchStringEmbedding, "EmbedDateUpdatedDesc");
            expect(builder.query.select).toContainEqual(
                "(1 / (POWER((\"c\".\"embedding\" <-> ARRAY[]::vector) + 0.01, 2))) + (EXP(-POWER(ABS(EXTRACT(EPOCH FROM NOW() - \"b\".\"updated_at\") / 3600), 0.25)) * 100) AS points",
            );
        });

        test("throws an error for unsupported sort option", () => {
            const objectType = "Routine";
            const translationObjectType = "RoutineTranslation" as any;
            const searchStringEmbedding = [];
            expect(() => {
                builder.embedPoints(translationObjectType, objectType, searchStringEmbedding, "UnsupportedOption" as any);
            }).toThrow("Unsupported sort option: UnsupportedOption");
        });
    });

    describe("equals function", () => {
        test("equals handles comparisons between fields", () => {
            const fieldRef1 = builder.field("Meeting", "userId");
            const fieldRef2 = builder.field("User", "id");
            expect(SqlBuilder.equals(fieldRef1, fieldRef2))
                .toBe("\"a\".\"userId\" = \"b\".\"id\"");
        });

        test("equals handles field to string literal comparisons", () => {
            const fieldRef = builder.field("Meeting", "userId");
            expect(SqlBuilder.equals(fieldRef, "hello"))
                .toBe("\"a\".\"userId\" = 'hello'");
        });

        test("equals handles field to numeric comparisons", () => {
            const fieldRef = builder.field("Meeting", "userId");
            expect(SqlBuilder.equals(fieldRef, 420))
                .toBe("\"a\".\"userId\" = 420");
        });

        test("equals handles field to boolean comparisons", () => {
            const fieldRef = builder.field("Meeting", "isActive");
            expect(SqlBuilder.equals(fieldRef, true))
                .toBe("\"a\".\"isActive\" = true");
        });
    });

    describe("and function", () => {
        test("combines two conditions correctly", () => {
            const condition1 = "\"age\" > 21";
            const condition2 = "\"status\" = 'active'";
            expect(SqlBuilder.and(condition1, condition2)).toBe("(\"age\" > 21) AND (\"status\" = 'active')");
        });

        test("combines a single condition correctly", () => {
            const condition = "\"status\" = 'active'";
            expect(SqlBuilder.and(condition)).toBe("\"status\" = 'active'");
        });

        test("correctly handles nested and and or", () => {
            const condition1 = "\"status\" = 'active'";
            const condition2 = "\"age\" > 21";
            const condition3 = "\"type\" = 'member'";
            expect(SqlBuilder.and(SqlBuilder.or(condition1, condition2), condition3)).toBe("((\"status\" = 'active') OR (\"age\" > 21)) AND (\"type\" = 'member')");
        });

        test("handles multiple nested conditions", () => {
            const condition1 = "\"status\" = 'active'";
            const condition2 = "\"age\" > 21";
            const condition3 = "\"type\" = 'member'";
            const condition4 = "\"expires\" < '2023-01-01'";
            expect(SqlBuilder.and(SqlBuilder.or(condition1, condition2), SqlBuilder.or(condition3, condition4))).toBe("((\"status\" = 'active') OR (\"age\" > 21)) AND ((\"type\" = 'member') OR (\"expires\" < '2023-01-01'))");
        });
    });

    describe("or function", () => {
        test("combines two conditions correctly", () => {
            const condition1 = "\"age\" > 21";
            const condition2 = "\"status\" = 'active'";
            expect(SqlBuilder.or(condition1, condition2)).toBe("(\"age\" > 21) OR (\"status\" = 'active')");
        });

        test("combines a single condition correctly", () => {
            const condition = "\"status\" = 'active'";
            expect(SqlBuilder.or(condition)).toBe("\"status\" = 'active'");
        });

        test("correctly handles nested and and or", () => {
            const condition1 = "\"status\" = 'active'";
            const condition2 = "\"age\" > 21";
            const condition3 = "\"type\" = 'member'";
            expect(SqlBuilder.or(SqlBuilder.and(condition1, condition2), condition3)).toBe("((\"status\" = 'active') AND (\"age\" > 21)) OR (\"type\" = 'member')");
        });

        test("handles multiple nested conditions", () => {
            const condition1 = "\"status\" = 'active'";
            const condition2 = "\"age\" > 21";
            const condition3 = "\"type\" = 'member'";
            const condition4 = "\"expires\" < '2023-01-01'";
            expect(SqlBuilder.or(SqlBuilder.and(condition1, condition2), SqlBuilder.and(condition3, condition4))).toBe("((\"status\" = 'active') AND (\"age\" > 21)) OR ((\"type\" = 'member') AND (\"expires\" < '2023-01-01'))");
        });
    });

    describe("not function", () => {
        test("negates a simple condition", () => {
            const condition = "\"status\" = 'active'";
            expect(SqlBuilder.not(condition)).toBe("NOT (\"status\" = 'active')");
        });

        test("negates a complex condition", () => {
            const condition = "\"age\" > 21 AND \"status\" = 'active'";
            expect(SqlBuilder.not(condition)).toBe("NOT (\"age\" > 21 AND \"status\" = 'active')");
        });

        test("negates a nested logical expression", () => {
            const condition = "(\"age\" > 21 OR \"status\" = 'active')";
            expect(SqlBuilder.not(condition)).toBe("NOT ((\"age\" > 21 OR \"status\" = 'active'))");
        });

        test("handles negation of a negated condition", () => {
            const condition = "NOT (\"status\" = 'active')";
            expect(SqlBuilder.not(condition)).toBe("NOT (NOT (\"status\" = 'active'))");
        });
    });
});
