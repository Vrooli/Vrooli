import { RunRoutineStepStatus } from "@shared/consts";
import { RunRoutineSearchInput, RunRoutineSortBy, RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { SelectWrap } from "../builders/types";
import { defaultPermissions } from "../utils";

// const shapeBase = (data: RunRoutineStepCreateInput | RunRoutineStepUpdateInput) => {
//     return {
//         id: data.id,
//         contextSwitches: data.contextSwitches ?? undefined,
//         timeElapsed: data.timeElapsed,
//     }
// }

const __typename = 'RunRoutineStep' as const;
const suppFields = [] as const;
export const RunRoutineStepModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineStepCreateInput,
    GqlUpdate: RunRoutineStepUpdateInput,
    GqlModel: RunRoutineStep,
    GqlPermission: {},
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.run_routine_stepUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_stepUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_stepGetPayload<SelectWrap<Prisma.run_routine_stepSelect>>,
    PrismaSelect: Prisma.run_routine_stepSelect,
    PrismaWhere: Prisma.run_routine_stepWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_step,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            run: 'RunRoutine',
            node: 'Node',
            subroutine: 'Routine',
        },
        prismaRelMap: {
            __typename,
            node: 'Node',
            runRoutine: 'RunRoutine',
            subroutine: 'RoutineVersion',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => {
                return {
                    // ...shapeBase(data),
                    // nodeId: data.nodeId,
                    // subroutineVersionId: data.subroutineVersionId,
                    // order: data.order,
                    // status: RunRoutineStepStatus.InProgress,
                    // step: data.step,
                    // name: data.name,
                } as any;
            },
            update: async ({ data, userData }) => {
                return {
                    // ...shapeBase(data),
                    // status: data.status ?? undefined,
                } as any
            }
        },
        yup: {} as any,
    },
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({
            id: true,
            runRoutine: 'RunRoutine',
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ['name'],
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