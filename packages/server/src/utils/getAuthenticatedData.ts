import { GqlModelType } from "@local/shared";
import { permissionsSelectHelper } from "../builders";
import { PrismaSelect } from "../builders/types";
import { CustomError, logger } from "../events";
import { getLogic } from "../getters";
import { PrismaType, SessionUserToken } from "../types";

/**
 * Given the primary keys of every object which needs to be authenticated, 
 * queries for all data required to perform authentication.
 */
export const getAuthenticatedData = async (
    idsByType: { [key in GqlModelType]?: string[] },
    prisma: PrismaType,
    userData: SessionUserToken | null,
): Promise<{ [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } }> => {
    // Initialize the return object
    const authDataById: { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } } = {};
    // For every type of object which needs to be authenticated, query for all data required to perform authentication
    for (const type of Object.keys(idsByType) as GqlModelType[]) {
        // Find validator and prisma delegate for this object type
        const { delegate, idField, validate } = getLogic(["delegate", "idField", "validate"], type, userData?.languages ?? ["en"], "getAuthenticatedData");

        // Query for data
        const where = { [idField]: { in: idsByType[type] } };
        let select: PrismaSelect | undefined;
        let data: any[];
        try {
            select = permissionsSelectHelper(validate.permissionsSelect, userData?.id ?? null, userData?.languages ?? ["en"]);
            data = await delegate(prisma).findMany({ where, select });
        } catch (error) {
            logger.error("getAuthenticatedData: findMany failed", { trace: "0453", error, type, select, where });
            throw new CustomError("0453", "InternalError", userData?.languages ?? ["en"], { objectType: type });
        }
        // Add data to return object
        for (const datum of data) {
            authDataById[datum[idField]] = { __typename: type, ...datum };
        }
    }
    // Return the data
    return authDataById;
};
