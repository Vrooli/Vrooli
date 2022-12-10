import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'Reminder' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.reminderSelect,
    Prisma.reminderGetPayload<SelectWrap<Prisma.reminderSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})