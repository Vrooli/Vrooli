import { PrismaType } from "types";
import { Role } from "../schema/types";
import { FormatConverter, addJoinTablesHelper, removeJoinTablesHelper, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { users: 'user' };
export const roleFormatter = (): FormatConverter<Role> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Role,
        'users': GraphQLModelType.User,
    },
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
    const prismaObject = prisma.role;
    const format = roleFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================