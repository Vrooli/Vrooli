import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const MeetingModel = ({
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Meeting' as GraphQLModelType,
    validate: {} as any,
})