import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ReminderListModel = ({
    delegate: (prisma: PrismaType) => prisma.reminder_list,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ReminderList' as GraphQLModelType,
    validate: {} as any,
})