import { combineQueries } from "../builders/combineQueries.js";
import { timeFrameToPrisma } from "../builders/timeFrame.js";
import { CountInputBase, PrismaDelegate } from "../builders/types.js";
import { visibilityBuilderPrisma } from "../builders/visibilityBuilder.js";
import { DbProvider } from "../db/provider.js";
import { ModelMap } from "../models/base/index.js";
import { CountHelperProps } from "./types.js";

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @returns The number of matching objects
 */
export async function countHelper<CountInput extends CountInputBase>({
    input,
    objectType,
    req,
    where,
    visibility,
}: CountHelperProps<CountInput>): Promise<number> {
    // Create query for created metric
    const createdQuery = timeFrameToPrisma("createdAt", input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma("updatedAt", input.updatedTimeFrame);
    // Create query for visibility, if supported
    const { query: visibilityQuery } = visibilityBuilderPrisma({ objectType, searchInput: input, req, visibility });
    // Count objects that match queries
    const delegate = DbProvider.get()[ModelMap.get(objectType).dbTable] as PrismaDelegate;
    return await delegate.count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery], { mergeMode: "strict" }),
    });
}
