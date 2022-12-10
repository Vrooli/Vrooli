import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'UserSchedule' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.user_scheduleSelect,
    Prisma.user_scheduleGetPayload<SelectWrap<Prisma.user_scheduleSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const UserScheduleModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})