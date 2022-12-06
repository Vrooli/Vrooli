import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const MeetingInviteModel = ({
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'MeetingInvite' as GraphQLModelType,
    validate: {} as any,
})