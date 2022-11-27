import { Role } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType } from "./types";

const formatter = (): Formatter<Role, any> => ({
    relationshipMap: {
        __typename: 'Role',
        assignees: 'User',
        organization: 'Organization',
    },
    joinMap: { assignees: 'user' },
})

export const RoleModel = ({
    delegate: (prisma: PrismaType) => prisma.role,
    format: formatter(),
    type: 'Role' as GraphQLModelType,
})