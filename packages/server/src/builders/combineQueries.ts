type CombineQueriesOptions = {
    mergeMode?: "strict" | "loose";
};

/**
 * Sensibly combines multiple Prisma query objects into a single Prisma query object.
 * 
 * NOTE 1: If queries have conflicting values (e.g. { isPrivate: true } and { isPrivate: false }), they 
 * will be combined into a query that will never return any results (e.g. { AND: [{ isPrivate: true }, { isPrivate: false }] }). 
 * This is done for safety reasons, as it's better to return no results than to return unexpected results.
 * 
 * NOTE 2: Inspect the tests for examples and to make sure the function works as expected.
 * 
 * @param queries Array of query objects to combine
 * @param options Used to specify how to combine the queries
 * @returns Combined query object, with all fields combined
 */
export function combineQueries(
    queries: (Record<string, unknown> | null | undefined)[],
    options?: CombineQueriesOptions,
): Record<string, unknown> {
    const conditions: Array<Record<string, unknown>> = [];
    // Default to "strict" mode for safety
    const optionsWithDefaults = { mergeMode: "strict", ...options } as const;

    for (const query of queries) {
        if (!query) continue;
        conditions.push(query);
    }

    const combined = mergeConditions(conditions, optionsWithDefaults);

    return combined;
}

function mergeConditions(
    conditions: Array<Record<string, unknown>>,
    options: CombineQueriesOptions,
): Record<string, unknown> {
    let result: Record<string, unknown> = {};

    for (const condition of conditions) {
        const merged = mergeTwoConditions(result, condition, options);
        result = merged.result;
    }

    return result;
}

function mergeTwoConditions(
    cond1: Record<string, unknown>,
    cond2: Record<string, unknown>,
    options: CombineQueriesOptions,
): { result: Record<string, unknown>, conflict?: boolean } {
    const result: Record<string, unknown> = {};
    let conflict = false;

    const keys = new Set([...Object.keys(cond1), ...Object.keys(cond2)]);

    for (const key of keys) {
        const val1 = cond1[key];
        const val2 = cond2[key];

        if (["AND", "OR", "NOT"].includes(key)) {
            // Handle logical operators
            const currValue = Array.isArray(val1) ? val1 : typeof val1 === "object" ? [val1] : [];
            const newValue = Array.isArray(val2) ? val2 : typeof val2 === "object" ? [val2] : [];

            // ORs get put in an AND with "strict" mode enabled
            if (key === "OR" && options.mergeMode === "strict") {
                const hasORsInAnd =
                    Array.isArray(result["AND"]) &&
                    result["AND"].some((item: Record<string, unknown>) =>
                        Object.prototype.hasOwnProperty.call(item, "OR") && Object.keys(item).length === 1,
                    );
                const shouldAddCurrValue = currValue.length > 0;
                const shouldAddNewValue = newValue.length > 0;
                // Only add ORs to AND if the total number of ORs is greater than 1
                if (!hasORsInAnd && !(shouldAddCurrValue && shouldAddNewValue)) {
                    result["OR"] = shouldAddCurrValue ? currValue : newValue;
                } else {
                    if (!result["AND"]) {
                        result["AND"] = [];
                    }
                    if (currValue.length > 0) {
                        result["AND"].push({ OR: currValue });
                    }
                    if (newValue.length > 0) {
                        result["AND"].push({ OR: newValue });
                    }
                }
            }
            // Otherwise, just merge the arrays
            else {
                if (!result[key]) {
                    result[key] = [];
                }
                if (currValue.length > 0) {
                    result[key].push(...currValue);
                }
                if (newValue.length > 0) {
                    result[key].push(...newValue);
                }
            }
        } else if (val1 === undefined) {
            result[key] = val2;
        } else if (val2 === undefined) {
            result[key] = val1;
        } else if (
            typeof val1 === "object" &&
            val1 !== null &&
            !Array.isArray(val1) &&
            typeof val2 === "object" &&
            val2 !== null &&
            !Array.isArray(val2)
        ) {
            // Both are objects, could be field conditions or nested objects
            const isFieldCond1 = isFieldCondition(val1 as Record<string, unknown>);
            const isFieldCond2 = isFieldCondition(val2 as Record<string, unknown>);

            if (isFieldCond1 && isFieldCond2) {
                const merged = mergeFieldConditions(val1 as Record<string, unknown>, val2 as Record<string, unknown>);
                if (merged.conflict) {
                    conflict = true;
                } else {
                    result[key] = merged.result;
                }
            } else {
                // Nested objects
                const merged = mergeTwoConditions(val1 as Record<string, unknown>, val2 as Record<string, unknown>, options);
                if (merged.conflict) {
                    result[key] = { AND: [val1, val2] };
                } else {
                    result[key] = merged.result;
                }
            }
        } else if (JSON.stringify(val1) === JSON.stringify(val2)) {
            result[key] = val1;
        } else {
            // Conflict
            conflict = true;
        }
    }

    if (conflict) {
        // Handle conflicts by extracting ORs to the top level
        const or1 = cond1.OR ? (Array.isArray(cond1.OR) ? cond1.OR : [cond1.OR]) : [];
        const or2 = cond2.OR ? (Array.isArray(cond2.OR) ? cond2.OR : [cond2.OR]) : [];
        const mergedORs = [...or1, ...or2];

        const cond1WithoutOR = { ...cond1 };
        delete cond1WithoutOR.OR;
        const cond2WithoutOR = { ...cond2 };
        delete cond2WithoutOR.OR;

        const andConditions: object[] = [];
        if (Object.keys(cond1WithoutOR).length > 0) {
            andConditions.push(cond1WithoutOR);
        }
        if (Object.keys(cond2WithoutOR).length > 0) {
            andConditions.push(cond2WithoutOR);
        }

        const newResult: Record<string, unknown> = {};
        if (mergedORs.length > 0) {
            newResult.OR = mergedORs;
        }
        if (andConditions.length > 0) {
            newResult.AND = andConditions;
        }
        return { result: newResult };
    } else {
        return { result };
    }
}

function mergeFieldConditions(
    condition1: Record<string, unknown>,
    condition2: Record<string, unknown>,
): { result?: Record<string, unknown>; conflict?: boolean } {
    const result: Record<string, unknown> = { ...condition1 };

    for (const [op, val] of Object.entries(condition2)) {
        if (op in result) {
            if (JSON.stringify(result[op]) !== JSON.stringify(val)) {
                // Conflict detected for the same operator with different values
                return { conflict: true };
            }
            // If values are equal, do nothing
        } else {
            result[op] = val;
        }
    }

    return { result };
}

function isFieldCondition(obj: Record<string, unknown>): boolean {
    const operators = new Set([
        "equals", "in", "notIn", "lt", "lte", "gt", "gte",
        "contains", "startsWith", "endsWith", "mode", "not",
    ]);
    return Object.keys(obj).every(key => operators.has(key));
}
