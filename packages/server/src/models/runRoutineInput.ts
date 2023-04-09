import { Prisma } from "@prisma/client";
import { MaxObjects, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputSearchInput, RunRoutineInputSortBy, RunRoutineInputUpdateInput } from '@shared/consts';
import { runRoutineInputValidation } from '@shared/validation';
import { RoutineVersionInputModel } from ".";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from '../utils';
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";

const __typename = 'RunRoutineInput' as const;
const suppFields = [] as const;
export const RunRoutineInputModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineInputCreateInput,
    GqlUpdate: RunRoutineInputUpdateInput,
    GqlModel: RunRoutineInput,
    GqlSearch: RunRoutineInputSearchInput,
    GqlSort: RunRoutineInputSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.run_routine_inputUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_inputUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_inputGetPayload<SelectWrap<Prisma.run_routine_inputSelect>>,
    PrismaSelect: Prisma.run_routine_inputSelect,
    PrismaWhere: Prisma.run_routine_inputWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_input,
    display: {
        select: () => ({
            id: true,
            input: selPad(RoutineVersionInputModel.display.select),
            runRoutine: selPad(RunRoutineModel.display.select),
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
            __typename,
            input: 'RoutineVersionInput',
            runRoutine: 'RunRoutine',
        },
        prismaRelMap: {
            __typename,
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
        yup: runRoutineInputValidation,
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
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: 'RunRoutine',
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ['data'],
        owner: (data, userId) => RunRoutineModel.validate!.owner(data.runRoutine as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => RunRoutineModel.validate!.isPublic(data.runRoutine as any, languages),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
        },
    },
})