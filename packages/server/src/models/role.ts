import { PrismaType } from "types";
import { Role } from "../schema/types";
import { FormatConverter, addJoinTablesHelper, removeJoinTablesHelper, ModelTypes } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { users: 'user' };
export const roleFormatter = (): FormatConverter<Role> => ({
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoleModel(prisma: PrismaType) {
    const model = ModelTypes.Role;
    const prismaObject = prisma.role;
    const format = roleFormatter();

    return {
        model,
        prismaObject,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================