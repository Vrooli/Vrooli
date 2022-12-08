import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.reminder_itemSelect,
    Prisma.reminder_itemGetPayload<SelectWrap<Prisma.reminder_itemSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderItemModel = ({
    delegate: (prisma: PrismaType) => prisma.reminder_item,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ReminderItem' as GraphQLModelType,
    validate: {} as any,
})