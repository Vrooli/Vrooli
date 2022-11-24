import { PrismaType } from "../types";
import { Member } from "../schema/types";
import { Formatter, GraphQLModelType } from "./types";

const formatter = (): Formatter<Member, any> => ({
    relationshipMap: {
        __typename: 'Member',
        organization: 'Organization',
        user: 'User',
    }
})

export const MemberModel = ({
    delegate: (prisma: PrismaType) => prisma.member,
    format: formatter(),
    type: 'Member' as GraphQLModelType,
})