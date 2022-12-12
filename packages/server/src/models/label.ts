import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Label, LabelCreateInput, LabelPermission, LabelSearchInput, LabelSortBy, LabelUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Searcher } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: LabelCreateInput,
    GqlUpdate: LabelUpdateInput,
    GqlRelCreate: LabelCreateInput,
    GqlRelUpdate: LabelUpdateInput,
    GqlModel: Label,
    GqlSearch: LabelSearchInput,
    GqlSort: LabelSortBy,
    GqlPermission: LabelPermission,
    PrismaCreate: Prisma.labelUpsertArgs['create'],
    PrismaUpdate: Prisma.labelUpsertArgs['update'],
    PrismaRelCreate: Prisma.labelCreateWithoutApisInput,
    PrismaRelUpdate: Prisma.labelUpdateWithoutApisInput,
    PrismaModel: Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>,
    PrismaSelect: Prisma.labelSelect,
    PrismaWhere: Prisma.labelWhereInput,
}

const __typename = 'Label' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
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
    countFields: ['apisCount', 'issuesCount', 'meetingsCount', 'notesCount', 'projectsCount', 'routinesCount', 'runProjectSchedulesCount', 'runRoutineSchedulesCount', 'userSchedulesCount'],
})

const searcher = (): Searcher<Model> => ({
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
    searchStringQuery: () => ({
        OR: [
            'labelWrapped',
            'transDescriptionWrapped'
        ]
    })
})

const displayer = (): Displayer<Model> => ({
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