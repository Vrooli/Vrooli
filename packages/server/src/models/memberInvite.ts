import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const MemberInviteModel = ({
    delegate: (prisma: PrismaType) => prisma.member_invite,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'MemberInvite' as GraphQLModelType,
    validate: {} as any,
})