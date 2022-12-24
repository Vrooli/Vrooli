import { RunRoutineStepStatus } from "@shared/consts";
import { RunRoutineSearchInput, RunRoutineSortBy, RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, ModelLogic, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineStepCreateInput,
    GqlUpdate: RunRoutineStepUpdateInput,
    GqlModel: RunRoutineStep,
    GqlPermission: any,
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.run_routine_stepUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_stepUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_stepGetPayload<SelectWrap<Prisma.run_routine_stepSelect>>,
    PrismaSelect: Prisma.run_routine_stepSelect,
    PrismaWhere: Prisma.run_routine_stepWhereInput,
}

const __typename = 'RunRoutineStep' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
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
})

const validator = (): Validator<Model> => ({
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
    profanityFields: ['name'],
    owner: (data) => RunRoutineModel.validate!.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunRoutineModel.validate!.isPublic(data.runRoutine as any, languages),
    visibility: {
        private: { runRoutine: { isPrivate: true } },
        public: { runRoutine: { isPrivate: false } },
        owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
    },
})

// const shapeBase = (data: RunRoutineStepCreateInput | RunRoutineStepUpdateInput) => {
//     return {
//         id: data.id,
//         contextSwitches: data.contextSwitches ?? undefined,
//         timeElapsed: data.timeElapsed,
//     }
// }

/**
 * Handles mutations of run steps
 */
const mutater = (): Mutater<Model> => ({
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
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunRoutineStepModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_step,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
    validate: validator(),
})