import { type ModelType, type SessionUser, validatePK } from "@vrooli/shared";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper.js";
import { type PrismaDelegate, type PrismaSelect } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";

export type AuthDataItem = { __typename: `${ModelType}`, [x: string]: any };
export type AuthDataById = { [id: string]: AuthDataItem };

/**
 * Given the primary keys of every object which needs to be authenticated, 
 * queries for all data required to perform authentication.
 */
export async function getAuthenticatedData(
    idsByType: { [key in ModelType]?: string[] },
    userData: Pick<SessionUser, "id"> | null,
): Promise<AuthDataById> {
    // Initialize the return object
    const authDataById: AuthDataById = {};
    // For every type of object which needs to be authenticated, query for all data required to perform authentication
    for (const type of Object.keys(idsByType) as ModelType[]) {
        // Find info for this object type
        const { dbTable, idField, validate } = ModelMap.getLogic(["dbTable", "idField", "validate"], type);
        const ids = idsByType[type] ?? [];
        // Build "where" clause
        let where: any = {};
        // If idField is "id", just use that
        if (idField === "id") {
            where = { id: { in: ids } };
        } else {
            // We may have "id" values mixed with idField values
            const validPks = ids.filter(validatePK);
            const otherIds = ids.filter(x => !validatePK(x));
            if (validPks.length) {
                where = { OR: [{ [idField]: { in: otherIds } }, { id: { in: validPks } }] };
            } else {
                where = { [idField]: { in: ids } };
            }
        }
        // Query for data
        let select: PrismaSelect | undefined;
        let data: any[];
        try {
            select = permissionsSelectHelper(validate().permissionsSelect, userData?.id ?? null);
            data = await (DbProvider.get()[dbTable] as PrismaDelegate).findMany({ where, select });
        } catch (error) {
            logger.error("getAuthenticatedData: findMany failed", { trace: "0453", error, type, select, where });
            throw new CustomError("0453", "InternalError", { objectType: type });
        }
        // Add data to return object
        for (const datum of data) {
            if (idField !== "id") authDataById[datum[idField]] = { __typename: type, ...datum };
            authDataById[datum.id] = { __typename: type, ...datum };
        }
    }
    // Return the data
    return authDataById;
}
