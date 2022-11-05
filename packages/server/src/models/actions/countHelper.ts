import { VisibilityType } from "../../schema/types";
import { combineQueries, getUserId, timeFrameToPrisma, visibilityBuilder } from "../builder";
import { CountInputBase, GraphQLModelType } from "../types";
import { CountHelperProps } from "./types";

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @returns The number of matching objects
 */
export async function countHelper<GraphQLModel, CountInput extends CountInputBase>({
    input,
    model,
    prisma,
    req,
    where,
    visibility = VisibilityType.Public,
}: CountHelperProps<GraphQLModel, CountInput>): Promise<number> {
    const userId = getUserId(req);
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create query for visibility, if supported
    let visibilityQuery: { [x: string]: any } | undefined;
    if ((['Organization', 'Project', 'Routine', 'Run', 'Standard']).includes(model.type)) {
        visibilityQuery = visibilityBuilder({ model, userId, visibility });
    }
    // Count objects that match queries
    return await (model.prismaObject(prisma) as any).count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery])
    });
}