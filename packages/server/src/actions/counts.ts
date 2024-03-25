import { VisibilityType } from "@local/shared";
import { getUser } from "../auth/request";
import { combineQueries } from "../builders/combineQueries";
import { timeFrameToPrisma } from "../builders/timeFrameToPrisma";
import { CountInputBase } from "../builders/types";
import { visibilityBuilder } from "../builders/visibilityBuilder";
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
    visibility = VisibilityType.Public,
}: CountHelperProps<CountInput>): Promise<number> {
    const userData = getUser(req.session);
    // Create query for created metric
    const createdQuery = timeFrameToPrisma("created_at", input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma("updated_at", input.updatedTimeFrame);
    // Create query for visibility, if supported
    const visibilityQuery = visibilityBuilder({ objectType, userData, visibility });
    // Count objects that match queries
    const delegate = ModelMap.get(objectType).delegate;
    return await delegate(prismaInstance).count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery]),
    });
}
