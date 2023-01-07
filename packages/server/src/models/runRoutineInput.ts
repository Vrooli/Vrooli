import { RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputSearchInput, RunRoutineInputSortBy, RunRoutineInputUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionInputModel } from ".";

const type = 'RunRoutineInput' as const;
const suppFields = [] as const;
export const RunRoutineInputModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineInputCreateInput,
    GqlUpdate: RunRoutineInputUpdateInput,
    GqlModel: RunRoutineInput,
    GqlSearch: RunRoutineInputSearchInput,
    GqlSort: RunRoutineInputSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.run_routine_inputUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_inputUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_inputGetPayload<SelectWrap<Prisma.run_routine_inputSelect>>,
    PrismaSelect: Prisma.run_routine_inputSelect,
    PrismaWhere: Prisma.run_routine_inputWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.run_routine_input,
    display: {
        select: () => ({
            id: true,
            input: padSelect(RoutineVersionInputModel.display.select),
            runRoutine: padSelect(RunRoutineModel.display.select),
        }),
        // Label combines runRoutine's label and input's label
        label: (select, languages) => {
            const runRoutineLabel = RunRoutineModel.display.label(select.runRoutine as any, languages);
            const inputLabel = RoutineVersionInputModel.display.label(select.input as any, languages);
            if (runRoutineLabel.length > 0) {
                return `${runRoutineLabel} - ${inputLabel}`;
            }
            return inputLabel;
        }
    },
    format: {
        gqlRelMap: {
            type,
            input: 'RoutineVersionInput',
            runRoutine: 'RunRoutine',
        },
        prismaRelMap: {
            type,
            input: 'RunRoutineInput',
            runRoutine: 'RunRoutine',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data }) => {
                return {
                    // id: data.id,
                    // data: data.data,
                    // input: { connect: { id: data.inputId } },
                } as any;
            },
            update: async ({ data }) => {
                return {
                    data: data.data
                }
            }
        },
        yup: {} as any,
    },
    search: {
        defaultSort: RunRoutineInputSortBy.DateUpdatedDesc,
        sortBy: RunRoutineInputSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            routineIds: true,
            standardIds: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ runRoutine: RunRoutineModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: (...params) => ({
            id: true,
            runRoutine: 'RunRoutine',
        }),
        permissionResolvers: ({ isAdmin, isPublic }) => ({
            canDelete: async () => isAdmin,
            canEdit: async () => isAdmin,
            canView: async () => isPublic,
        }),
        profanityFields: ['data'],
        owner: (data) => RunRoutineModel.validate!.owner(data.runRoutine as any),
        isDeleted: () => false,
        isPublic: (data, languages) => RunRoutineModel.validate!.isPublic(data.runRoutine as any, languages),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
        },
    },
})