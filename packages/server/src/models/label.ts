import { Prisma } from "@prisma/client";
import { searchStringBuilder } from "../builders";
import { SelectWrap } from "../builders/types";
import { Label, LabelSearchInput, LabelSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Searcher } from "./types";

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

const searcher = (): Searcher<
    LabelSearchInput,
    LabelSortBy,
    Prisma.labelWhereInput
> => ({
    defaultSort: LabelSortBy.DateUpdatedDesc,
    sortBy: LabelSortBy,
    searchFields: [
        'createdTimeFrame',
        'label',
        'ownedByOrganizationId',
        'ownedByUserId',
        'translationLanguages',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: (params) => ({
        OR: searchStringBuilder(['label', 'translationsDescription'], params),
    }),
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
    search: searcher(),
    validate: {} as any,
})