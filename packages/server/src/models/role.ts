import { Role } from "../schema/types";
import { PrismaType } from "../types";
import { addJoinTablesHelper, removeJoinTablesHelper } from "./builder";
import { Formatter, GraphQLModelType } from "./types";

const joinMapper = { assignees: 'user' };
const formatter = (): Formatter<Role, any> => ({
    relationshipMap: {
        __typename: 'Role',
        assignees: 'User',
        organization: 'Organization',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
})

export const RoleModel = ({
    delegate: (prisma: PrismaType) => prisma.role,
    format: formatter(),
    type: 'Role' as GraphQLModelType,
})