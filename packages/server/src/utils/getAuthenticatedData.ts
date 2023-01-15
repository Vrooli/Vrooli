import { getLogic } from "../getters";
import { GqlModelType, SessionUser } from '@shared/consts';
import { PrismaType } from "../types";
import { permissionsSelectHelper } from "../builders";

/**
 * Given the IDs of every object which needs to be authenticated, queries for all data required to perform authentication.
 */
export const getAuthenticatedData = async (
    idsByType: { [key in GqlModelType]?: string[] },
    prisma: PrismaType,
    userData: SessionUser | null,
): Promise<{ [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } }> => {
    console.log('getAuthenticatedData start', JSON.stringify(idsByType), '\n\n');
    // Initialize the return object
    const authDataById: { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } } = {};
    // For every type of object which needs to be authenticated, query for all data required to perform authentication
    for (const type of Object.keys(idsByType) as GqlModelType[]) {
        // Find validator and prisma delegate for this object type
        const { delegate, validate } = getLogic(['delegate', 'validate'], type, userData?.languages ?? ['en'], 'getAuthenticatedData');
        // Query for data
        const data = await delegate(prisma).findMany({
            where: { id: { in: idsByType[type] } },
            select: permissionsSelectHelper(validate.permissionsSelect, userData?.id ?? null, userData?.languages ?? ['en']),
        });
        // Add data to return object
        for (const datum of data) {
            authDataById[datum.id] = { __typename: type, ...datum }
        }
    }
    console.log('getAuthenticatedData end', JSON.stringify(authDataById), '\n\n');
    // Return the data
    return authDataById;
}