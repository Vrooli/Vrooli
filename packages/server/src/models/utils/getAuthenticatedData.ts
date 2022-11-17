import { PrismaType } from "../../types";
import { GraphQLModelType } from "../types";
import { getDelegate } from "./getDelegate";
import { getValidator } from "./getValidator";

/**
 * Given the IDs of every object which needs to be authenticated, queries for all data required to perform authentication.
 */
export const getAuthenticatedData = async (
    idsByType: { [key in GraphQLModelType]?: string[] },
    prisma: PrismaType,
    userId: string | null,
): Promise<{ [id: string]: { __typename: GraphQLModelType, [x: string]: any } }> => {
    // Initialize the return object
    const authDataById: { [id: string]: { __typename: GraphQLModelType, [x: string]: any } } = {};
    // For every type of object which needs to be authenticated, query for all data required to perform authentication
    for (const type of Object.keys(idsByType) as GraphQLModelType[]) {
        // Find validator and prisma delegate for this object type
        const validator = getValidator(type, 'getAuthenticatedData');
        const prismaDelegate = getDelegate(type, prisma, 'getAuthenticatedData');
        // Query for data
        const data = await prismaDelegate.findMany({
            where: { id: { in: idsByType[type] } },
            select: validator.permissionsSelect(userId),
        });
        // Add data to return object
        for (const datum of data) {
            authDataById[datum.id] = {
                __typename: type,
                ...datum,
            }
        }
    }
    // Return the data
    return authDataById;
}