import { PrismaType } from "../../types";
import { Role } from "../../schema/types";
import { FormatConverter, addJoinTablesHelper, removeJoinTablesHelper } from "./base";
import { GraphQLModelType } from ".";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { assignees: 'user' };
export const roleFormatter = (): FormatConverter<Role, any> => ({
    relationshipMap: {
        '__typename': 'Role',
        'assignees': 'User',
        'organization': 'Organization',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const RoleModel = ({
    prismaObject: (prisma: PrismaType) => prisma.role,
    format: roleFormatter(),
    type: 'Role' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================