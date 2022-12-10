import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";
import { UserScheduleModel } from "./userSchedule";

const __typename = 'ReminderList' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.reminder_listSelect,
    Prisma.reminder_listGetPayload<SelectWrap<Prisma.reminder_listSelect>>
> => ({
    select: () => ({ id: true, userSchedule: { select: UserScheduleModel.display.select() } }),
    // Label is schedule's label
    label: (select, languages) =>  UserScheduleModel.display.label(select.userSchedule as any, languages),
})

export const ReminderListModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_list,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})