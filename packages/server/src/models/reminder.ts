import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ReminderModel = ({
    delegate: (prisma: PrismaType) => prisma.reminder,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Reminder' as GraphQLModelType,
    validate: {} as any,
})