import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Label } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

const __typename = 'Label' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Label, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        apis: 'Api',
        issues: 'Issue',
        meetings: 'Meeting',
        notes: 'Note',
        projects: 'Project',
        routines: 'Routine',
        runProjectSchedules: 'RunProjectSchedule',
        runRoutineSchedules: 'RunRoutineSchedule',
        userSchedules: 'UserSchedule',
    },
    joinMap: { starredBy: 'user' },
    countFields: ['apisCount', 'issuesCount', 'meetingsCount', 'notesCount', 'projectsCount', 'routinesCount', 'runProjectSchedulesCount', 'runRoutineSchedulesCount', 'userSchedulesCount'],
})

const displayer = (): Displayer<
    Prisma.labelSelect,
    Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>
> => ({
    select: () => ({ id: true, label: true }),
    label: (select) => select.label,
})

export const LabelModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.label,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})