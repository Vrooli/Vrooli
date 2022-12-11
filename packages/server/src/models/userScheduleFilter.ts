import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { TagModel } from "./tag";
import { Displayer } from "./types";

const __typename = 'UserScheduleFilter' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.user_schedule_filterSelect,
    Prisma.user_schedule_filterGetPayload<SelectWrap<Prisma.user_schedule_filterSelect>>
> => ({
    select: () => ({ id: true, tag: { select: TagModel.display.select() } }),
    label: (select, languages) => select.tag ? TagModel.display.label(select.tag as any, languages) : '',
})

export const UserScheduleFilterModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule_filter,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})