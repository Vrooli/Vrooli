// AI_CHECK: TYPE_SAFETY=phase2-auth-data | LAST: 2025-07-04 - Enhanced type safety for authenticated data retrieval
import { type ModelType, type SessionUser, validatePK } from "@vrooli/shared";
import { hasId, hasTypename } from "./typeGuards.js";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper.js";
import { type PrismaDelegate, type PrismaSelect } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";

export type AuthDataItem = { __typename: `${ModelType}`, id: string, [x: string]: unknown };
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
        const { dbTable, idField, validate } = ModelMap.getLogic(["dbTable", "idField", "validate"], type, true, "getAuthenticatedData");
        const ids = idsByType[type] ?? [];
        // Build "where" clause
        let where: Record<string, unknown> = {};
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
        let data: Record<string, unknown>[];
        try {
            const validator = validate();
            if (!validator || !validator.permissionsSelect) {
                throw new CustomError("0454", "InternalError", { type });
            }
            select = permissionsSelectHelper(validator.permissionsSelect, userData?.id ?? null);
            const dbResults = await (DbProvider.get()[dbTable] as PrismaDelegate).findMany({ where, select });
            data = dbResults.map(item => ({ ...item, select }));
        } catch (error) {
            logger.error("getAuthenticatedData: findMany failed", { trace: "0453", error, type, select, where });
            throw new CustomError("0453", "InternalError", { objectType: type });
        }
        // Add data to return object
        for (const datum of data) {
            if (idField !== "id" && typeof datum[idField] === "string") {
                authDataById[datum[idField] as string] = { __typename: type, id: datum[idField] as string, ...datum } as AuthDataItem;
            }
            if (typeof datum.id === "string") {
                authDataById[datum.id] = { __typename: type, id: datum.id, ...datum } as AuthDataItem;
            }
        }
    }
    // Return the data
    return authDataById;
}
