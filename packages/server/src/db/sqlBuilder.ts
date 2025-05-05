import { ModelType } from "@local/shared";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { EmbedSortOption } from "../utils/embeddings/cache.js";

export type SQLQuery = {
    select: (SelectClause | string)[];
    from: FromClause;
    joins: JoinClause[];
    where: WhereCondition[];
    groupBy?: GroupByClause[];
    having?: WhereCondition[];
    orderBy: string[];
    limit?: number;
    offset?: number;
};

export type SelectClause = {
    field: string;
    alias?: string;
    tableAlias?: string;
};

export type FromClause = {
    /** The db table name */
    table: string;
    /** The alias used to reference the table */
    alias: string;
};

export type JoinClause = {
    type: JoinType | `${JoinType}`;
    /** The db table name */
    table: string;
    /** The alias used to reference the table */
    alias: string;
    on: string;
};

export type WhereCondition = {
    condition: string;
    parameters?: any[];
};

export type GroupByClause = { field: string; tableAlias?: string; };

export type OrderByClause = { field: string; direction: "ASC" | "DESC"; tableAlias?: string; };

/** The type of join operation to perform. Picture a Venn diagram. */
export enum JoinType {
    /** Records matching values in both tables. The inner part of a Venn diagram */
    INNER = "INNER",
    /** All records from the left table, and matching records from the right table. The left circle of a Venn diagram */
    LEFT = "LEFT",
    /** All records from the right table, and matching records from the left table. The right circle of a Venn diagram */
    RIGHT = "RIGHT",
    /** All records from both tables. The entire Venn diagram */
    FULL = "FULL"
}

type FieldReference = {
    tableAlias: string;
    column: string;
};

export class SqlBuilder {
    query: SQLQuery;
    private tableToAliasMap: { [obectType in ModelType | `${ModelType}`]?: string };

    constructor(rootObjectType: ModelType | `${ModelType}`) {
        this.tableToAliasMap = {};
        const mainTable = ModelMap.get(rootObjectType).dbTable;

        this.query = {
            select: [],
            from: { table: mainTable, alias: this.getAlias(rootObjectType) },
            joins: [],
            where: [],
            orderBy: [],
        };
    }

    /**
     * Finds the alias representing an object type. 
     * Creates a new alias if one does not exist.
     */
    getAlias(objectType: ModelType | `${ModelType}`): string {
        const dbTable = ModelMap.get(objectType).dbTable;
        if (!this.tableToAliasMap[dbTable]) {
            if (Object.keys(this.tableToAliasMap).length >= 26) {
                throw new CustomError("0422", "InternalError", { message: "Alias overflow: More than 26 table aliases are not supported." });
            }
            // Move across unicode characters, starting at "a", to create a new alias
            const nextChar = String.fromCharCode("a".charCodeAt(0) + Object.keys(this.tableToAliasMap).length);
            this.tableToAliasMap[dbTable] = nextChar;
        }
        return this.tableToAliasMap[dbTable] as string;
    }

    addSelect(objectType: ModelType | `${ModelType}`, field: string, alias?: string) {
        const tableAlias = this.getAlias(objectType);
        this.query.select.push({ field, alias, tableAlias });
    }

    addSelectRaw(sql: string) {
        this.query.select.push(sql);
    }

    addJoin(
        objectType: ModelType | `${ModelType}`,
        type: JoinType | `${JoinType}`,
        on: string,
    ) {
        const joinAlias = this.getAlias(objectType);
        const table = ModelMap.get(objectType).dbTable;
        this.query.joins.push({ type, table, alias: joinAlias, on });
    }

    addWhere(condition: string) {
        this.query.where.push({ condition });
    }

    addOrderBy(objectType: ModelType | `${ModelType}`, fieldName: string, direction: "DESC" | "ASC") {
        const tableAlias = this.getAlias(objectType);
        const orderClause = `"${tableAlias}"."${fieldName}" ${direction}`;
        this.query.orderBy.push(orderClause);
    }

    addOrderByRaw(sql: string) {
        this.query.orderBy.push(sql);
    }

    setLimit(count: number) {
        if (isNaN(count) || !Number.isInteger(count) || count < 1 || count > 1000) {
            throw new Error("Limit must be a positive integer between 1 and 1000");
        }
        this.query.limit = count;
    }

    setOffset(offset: number) {
        if (isNaN(offset) || !Number.isInteger(offset) || offset < 0) {
            throw new Error("Offset must be a non-negative integer");
        }
        this.query.offset = offset;
    }

    field(objectType: ModelType | `${ModelType}`, column: string): FieldReference {
        const tableAlias = this.getAlias(objectType);
        return { tableAlias, column };
    }

    /**
     * Generates SQL for selecting a "points" calculation
     */
    embedPoints(
        translationObjectType: ModelType | `${ModelType}`, // Table where embeddings are stored
        objectType: ModelType | `${ModelType}`, // Table where additional field (i.e. bookmarks or date field) is stored
        searchStringEmbedding: number[],
        sortOption: EmbedSortOption | `${EmbedSortOption}`,
    ) {
        const tableAlias = this.getAlias(objectType);
        const translationTableAlias = this.getAlias(translationObjectType);
        // Convert the search string embedding to a Postgres array, with reduced precision
        const embeddingArray = `ARRAY[${searchStringEmbedding.map(e => e.toFixed(6)).join(", ")}]`;
        // Small number to avoid division by zero (happens if search string embedding exactly matches row's embedding)
        const e = 0.01;
        // Euclidean distance of search string embedding from row's embedding
        const distance = `"${translationTableAlias}"."embedding" <-> ${embeddingArray}::vector`;
        // Distance converted to be useful for scoring
        const inverseDistanceWeighted = `(1 / (POWER((${distance}) + ${e}, 2)))`;
        // The rest of the score varies based on the sort option. 
        // These formulas are based on experimentation in https://github.com/Vrooli/SearchEmbeddingScores
        let scoreExpression = "";
        switch (sortOption) {
            case "EmbedTopAsc":
                scoreExpression = `${inverseDistanceWeighted} + (EXP(-"${tableAlias}"."bookmarks" / 5) * 20)`;
                break;
            case "EmbedTopDesc":
                scoreExpression = `${inverseDistanceWeighted} + (LN("${tableAlias}"."bookmarks" + 1) * 2)`;
                break;
            case "EmbedDateCreatedAsc":
                scoreExpression = `${inverseDistanceWeighted} + (LOG(1 + POWER(ABS(EXTRACT(EPOCH FROM NOW() - "${tableAlias}"."createdAt") / 3600), 0.25)) * 1)`;
                break;
            case "EmbedDateCreatedDesc":
                scoreExpression = `${inverseDistanceWeighted} + (EXP(-POWER(ABS(EXTRACT(EPOCH FROM NOW() - "${tableAlias}"."createdAt") / 3600), 0.25)) * 100)`;
                break;
            case "EmbedDateUpdatedAsc":
                scoreExpression = `${inverseDistanceWeighted} + (LOG(1 + POWER(ABS(EXTRACT(EPOCH FROM NOW() - "${tableAlias}"."updatedAt") / 3600), 0.25)) * 1)`;
                break;
            case "EmbedDateUpdatedDesc":
                scoreExpression = `${inverseDistanceWeighted} + (EXP(-POWER(ABS(EXTRACT(EPOCH FROM NOW() - "${tableAlias}"."updatedAt") / 3600), 0.25)) * 100)`;
                break;
            default:
                throw new Error(`Unsupported sort option: ${sortOption}`);
        }

        const select = `${scoreExpression} AS points`;
        this.query.select.push(select);
    }

    // Recursive function to handle nested queries and relationships
    buildQueryFromPrisma(filter: any, depth = 0, parentAlias?: string) {
        // Here you would recursively process the Prisma filter object
        // translating it into SQL components using the helper methods above.
        // This is a complex function that would need to handle Prisma's query syntax.
    }

    serialize(): string {
        let selects = "*";
        if (this.query.select.length > 0) {
            selects = this.query.select.map(s => {
                if (typeof s === "string") {
                    // Handle raw SQL strings directly
                    return s;
                } else {
                    // Handle structured field selections
                    return `"${s.tableAlias}"."${s.field}"${s.alias ? " AS \"" + s.alias + "\"" : ""}`;
                }
            }).join(", ");
        }
        const from = `FROM "${this.query.from.table}" AS "${this.query.from.alias}"`;
        const joins = this.query.joins.map(j => `${j.type} JOIN "${j.table}" AS "${j.alias}" ON ${j.on}`).join(" ");
        const where = this.query.where.length > 0 ? `WHERE ${this.query.where.map(w => w.condition).join(" AND ")}` : "";
        const orderBy = this.query.orderBy.length > 0 ? `ORDER BY ${this.query.orderBy.join(", ")}` : "";
        const limit = this.query.limit ? `LIMIT ${this.query.limit}` : "";
        const offset = this.query.offset ? `OFFSET ${this.query.offset}` : "";

        let query = `SELECT ${selects} ${from}`;
        if (joins) query += ` ${joins}`;
        if (where) query += ` ${where}`;
        if (orderBy) query += ` ${orderBy}`;
        if (limit) query += ` ${limit}`;
        if (offset) query += ` ${offset}`;
        query += ";";

        return query;
    }

    static equals(leftOperand: FieldReference | string, rightOperand: FieldReference | string | number | boolean): string {
        // Helper function to format field references or literals
        function formatOperand(operand: FieldReference | string | number | boolean): string {
            if (typeof operand === "object" && "tableAlias" in operand && "column" in operand) {
                return `"${operand.tableAlias}"."${operand.column}"`;
            } else if (typeof operand === "string") {
                return `'${operand.replace(/'/g, "''")}'`;
            }
            return operand.toString();
        }

        return `${formatOperand(leftOperand)} = ${formatOperand(rightOperand)}`;
    }

    static and(...conditions: string[]): string {
        if (conditions.length > 1) {
            return "(" + conditions.join(") AND (") + ")";
        }
        return conditions.join(""); // For a single condition, no need for parentheses
    }

    static or(...conditions: string[]): string {
        if (conditions.length > 1) {
            return "(" + conditions.join(") OR (") + ")";
        }
        return conditions.join(""); // For a single condition, no need for parentheses
    }

    static not(condition: string): string {
        return `NOT (${condition})`;
    }
}
