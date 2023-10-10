import { GqlModelType, uuidValidate } from "@local/shared";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper";
import { PrismaSelect } from "../builders/types";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { ModelMap } from "../models/base";
import { PrismaType, SessionUserToken } from "../types";

export type AuthDataById = { [id: string]: { __typename: `${GqlModelType}`, [x: string]: any } };

/**
 * Given the primary keys of every object which needs to be authenticated, 
 * queries for all data required to perform authentication.
 */
export const getAuthenticatedData = async (
    idsByType: { [key in GqlModelType]?: string[] },
    prisma: PrismaType,
    userData: SessionUserToken | null,
): Promise<AuthDataById> => {
    // Initialize the return object
    const authDataById: AuthDataById = {};
    // For every type of object which needs to be authenticated, query for all data required to perform authentication
    for (const type of Object.keys(idsByType) as GqlModelType[]) {
        // Find validator and prisma delegate for this object type
        const { delegate, idField, validate } = ModelMap.getLogic(["delegate", "idField", "validate"], type);
        const ids = idsByType[type] ?? [];
        // Build "where" clause
        let where: any = {};
        // If idField is "id", just use that
        if (idField === "id") {
            where = { id: { in: ids } };
        } else {
            // We may have "id" values mixed with idField values
            const uuids = ids.filter(uuidValidate);
            const otherIds = ids.filter(x => !uuidValidate(x));
            if (uuids.length) {
                where = { OR: [{ [idField]: { in: otherIds } }, { id: { in: uuids } }] };
            } else {
                where = { [idField]: { in: ids } };
            }
        }
        // Query for data
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
            if (idField !== "id") authDataById[datum[idField]] = { __typename: type, ...datum };
            authDataById[datum.id] = { __typename: type, ...datum };
        }
    }
    // Return the data
    return authDataById;
};
