import { SessionService } from "../auth/session";
import { combineQueries } from "../builders/combineQueries";
import { timeFrameToPrisma } from "../builders/timeFrame";
import { CountInputBase, PrismaDelegate } from "../builders/types";
import { visibilityBuilderPrisma } from "../builders/visibilityBuilder";
import { prismaInstance } from "../db/instance";
import { ModelMap } from "../models/base";
import { CountHelperProps } from "./types";

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
    const userData = SessionService.getUser(req.session);
    // Create query for created metric
    const createdQuery = timeFrameToPrisma("created_at", input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma("updated_at", input.updatedTimeFrame);
    // Create query for visibility, if supported
    const { query: visibilityQuery } = visibilityBuilderPrisma({ objectType, searchInput: input, userData, visibility });
    // Count objects that match queries
    const delegate = prismaInstance[ModelMap.get(objectType).dbTable] as PrismaDelegate;
    return await delegate.count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery], { mergeMode: "strict" }),
    });
}
