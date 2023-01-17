import { VisibilityType } from '@shared/consts';
import { CountHelperProps } from "./types";
import { CountInputBase } from "../builders/types";
import { getUser } from "../auth";
import { combineQueries, timeFrameToPrisma, visibilityBuilder } from "../builders";
import { getLogic } from "../getters";

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @returns The number of matching objects
 */
export async function countHelper<CountInput extends CountInputBase>({
    input,
    objectType,
    prisma,
    req,
    where,
    visibility = VisibilityType.Public,
}: CountHelperProps<CountInput>): Promise<number> {
    const userData = getUser(req);
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create query for visibility, if supported
    const visibilityQuery = visibilityBuilder({ objectType, userData, visibility });
    // Count objects that match queries
    const { delegate } = getLogic(['delegate'], objectType, userData?.languages ?? ['en'], 'countHelper')
    return await delegate(prisma).count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery])
    });
}