import { Role } from "../schema/types";
import { PrismaType } from "../types";
import { addJoinTablesHelper, removeJoinTablesHelper } from "./builder";
import { FormatConverter, GraphQLModelType } from "./types";

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

export const RoleModel = ({
    prismaObject: (prisma: PrismaType) => prisma.role,
    format: roleFormatter(),
    type: 'Role' as GraphQLModelType,
})