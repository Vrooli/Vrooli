import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.reminderSelect,
    Prisma.reminderGetPayload<SelectWrap<Prisma.reminderSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderModel = ({
    delegate: (prisma: PrismaType) => prisma.reminder,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Reminder' as GraphQLModelType,
    validate: {} as any,
})