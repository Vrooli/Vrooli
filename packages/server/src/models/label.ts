import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput, LabelYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const type = 'Label' as const;
type Permissions = Pick<LabelYou, 'canDelete' | 'canEdit'>;
const suppFields = ['you.canDelete', 'you.canEdit'] as const;
export const LabelModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: LabelCreateInput,
    GqlUpdate: LabelUpdateInput,
    GqlModel: Label,
    GqlSearch: LabelSearchInput,
    GqlSort: LabelSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.labelUpsertArgs['create'],
    PrismaUpdate: Prisma.labelUpsertArgs['update'],
    PrismaModel: Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>,
    PrismaSelect: Prisma.labelSelect,
    PrismaWhere: Prisma.labelWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.label,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            type,
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
        prismaRelMap: {
            type,
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
        joinMap: { 
            apis: 'labelled', 
            issues: 'labelled',
            meetings: 'labelled',
            notes: 'labelled',
            projects: 'labelled',
            routines: 'labelled',
            runProjectSchedules: 'labelled',
            runRoutineSchedules: 'labelled',
            userSchedules: 'labelled',
        },
        countFields: {
            apisCount: true,
            issuesCount: true,
            meetingsCount: true,
            notesCount: true,
            projectsCount: true,
            smartContractsCount: true,
            standardsCount: true,
            routinesCount: true,
            runProjectSchedulesCount: true,
            runRoutineSchedulesCount: true,
            translationsCount: true,
            userSchedulesCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                }
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: LabelSortBy.DateUpdatedDesc,
        sortBy: LabelSortBy,
        searchFields: {
            createdTimeFrame: true,
            label: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'labelWrapped',
                'transDescriptionWrapped'
            ]
        })
    },
    validate: {} as any,
})