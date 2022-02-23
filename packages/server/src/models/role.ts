import { role } from "@prisma/client";
import { RecursivePartial, PrismaType } from "types";
import { Role } from "../schema/types";
import { FormatConverter, addJoinTables, removeJoinTables, MODEL_TYPES } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
export const roleFormatter = (): FormatConverter<Role, role> => {
    const joinMapper = {
        users: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Role>): RecursivePartial<role> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<role>): RecursivePartial<Role> => removeJoinTables(obj, joinMapper)
    }
}

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoleModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Role;
    const format = roleFormatter();

    return {
        prisma,
        model,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================