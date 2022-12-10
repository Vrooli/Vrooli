import { stepsCreate, stepsUpdate } from "@shared/validation";
import { RunStepStatus } from "@shared/consts";
import { RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { SelectWrap } from "../builders/types";

const __typename = 'RunRoutineStep' as const;

const suppFields = [] as const;
const formatter = (): Formatter<RunRoutineStep, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        run: 'RunRoutine',
        node: 'Node',
        subroutine: 'Routine',
    },
})

const validator = (): Validator<
    RunRoutineStepCreateInput,
    RunRoutineStepUpdateInput,
    Prisma.run_routine_stepGetPayload<SelectWrap<Prisma.run_routine_stepSelect>>,
    any,
    Prisma.run_routine_stepSelect,
    Prisma.run_routine_stepWhereInput,
    false,
    false
> => ({
    validateMap: {
        __typename: 'RunRoutine',
        node: 'Node',
        runRoutine: 'RunRoutine',
        subroutine: 'Routine',
    },
    isTransferable: false,
    maxObjects: 100000,
    permissionsSelect: (...params) => ({
        runRoutine: { select: RunRoutineModel.validate.permissionsSelect(...params) }
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canView', async () => isPublic],
    ]),
    profanityFields: ['name'],
    owner: (data) => RunRoutineModel.validate.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunRoutineModel.validate.isPublic(data.runRoutine as any, languages),
    visibility: {
        private: { runRoutine: { isPrivate: true } },
        public: { runRoutine: { isPrivate: false } },
        owner: (userId) => ({ runRoutine: RunRoutineModel.validate.visibility.owner(userId) }),
    },
})

const shapeBase = (data: RunRoutineStepCreateInput | RunRoutineStepUpdateInput) => {
    return {
        id: data.id,
        contextSwitches: data.contextSwitches ?? undefined,
        timeElapsed: data.timeElapsed,
    }
}

/**
 * Handles mutations of run steps
 */
const mutater = (): Mutater<
    RunRoutineStep,
    false,
    false,
    { graphql: RunRoutineStepCreateInput, db: Prisma.run_routine_stepCreateWithoutRunRoutineInput },
    { graphql: RunRoutineStepUpdateInput, db: Prisma.run_routine_stepUpdateWithoutRunRoutineInput }
> => ({
    shape: {
        relCreate: async ({ data, userData }) => {
            return {
                ...shapeBase(data),
                nodeId: data.nodeId,
                subroutineVersionId: data.subroutineVersionId,
                order: data.order,
                status: RunStepStatus.InProgress,
                step: data.step,
                name: data.name,
            }
        },
        relUpdate: async ({ data, userData }) => {
            return {
                ...shapeBase(data),
                status: data.status ?? undefined,
            }
        }
    },
    yup: { create: stepsCreate, update: stepsUpdate },
})

const displayer = (): Displayer<
    Prisma.run_routine_stepSelect,
    Prisma.run_routine_stepGetPayload<SelectWrap<Prisma.run_routine_stepSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunRoutineStepModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_step,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    validate: validator(),
})