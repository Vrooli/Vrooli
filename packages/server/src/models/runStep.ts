import { stepsCreate, stepsUpdate } from "@shared/validation";
import { RunStepStatus } from "@shared/consts";
import { RunStep, RunStepCreateInput, RunStepUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { RunModel } from "./run";
import { OrganizationModel } from "./organization";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";

const formatter = (): Formatter<RunStep, any> => ({
    relationshipMap: {
        __typename: 'RunStep',
        run: 'RunRoutine',
        node: 'Node',
        subroutine: 'Routine',
    },
})

const validator = (): Validator<
    RunStepCreateInput,
    RunStepUpdateInput,
    RunStep,
    Prisma.run_routine_stepGetPayload<{ select: { [K in keyof Required<Prisma.run_routine_stepSelect>]: true } }>,
    any,
    Prisma.run_routine_stepSelect,
    Prisma.run_routine_stepWhereInput
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
        runRoutine: { select: RunModel.validate.permissionsSelect(...params) }
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canView', async () => isPublic],
    ]),
    profanityFields: ['title'],
    owner: (data) => RunModel.validate.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunModel.validate.isPublic(data.runRoutine as any, languages),
    visibility: {
        private: { runRoutine: { isPrivate: true } },
        public: { runRoutine: { isPrivate: false } },
        owner: (userId) => ({ runRoutine: RunModel.validate.visibility.owner(userId) }),
    },
})

const shapeBase = (data: RunStepCreateInput | RunStepUpdateInput) => {
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
    RunStep,
    false,
    false,
    { graphql: RunStepCreateInput, db: Prisma.run_routine_stepCreateWithoutRunRoutineInput },
    { graphql: RunStepUpdateInput, db: Prisma.run_routine_stepUpdateWithoutRunRoutineInput }
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
                title: data.title,
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
    Prisma.run_routine_stepGetPayload<{ select: { [K in keyof Required<Prisma.run_routine_stepSelect>]: true } }>
> => ({
    select: { id: true, title: true },
    label: (select) => select.title,
})

export const RunStepModel = ({
    delegate: (prisma: PrismaType) => prisma.run_routine_step,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'RunStep' as GraphQLModelType,
    validate: validator(),
})