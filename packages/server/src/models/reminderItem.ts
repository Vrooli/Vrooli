import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'ReminderItem' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.reminder_itemSelect,
    Prisma.reminder_itemGetPayload<SelectWrap<Prisma.reminder_itemSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderItemModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_item,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})