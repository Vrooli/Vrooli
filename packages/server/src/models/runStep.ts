import { stepsCreate, stepsUpdate } from "@shared/validation";
import { RunStepStatus } from "@shared/consts";
import { relationshipBuilderHelper, RelationshipTypes } from "./builder";
import { RunStep, RunStepCreateInput, RunStepUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

export const runStepFormatter = (): FormatConverter<RunStep, any> => ({
    relationshipMap: {
        __typename: 'RunStep',
        run: 'RunRoutine',
        node: 'Node',
        subroutine: 'Routine',
    },
})

export const runStepVerifier = () => ({
    profanityCheck(data: (RunStepCreateInput | RunStepUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.title));
    },
})

/**
 * Handles mutations of run steps
 */
export const runStepMutater = (prisma: PrismaType): Mutater<RunStep> => ({
    shapeBase(userId: string, data: RunStepCreateInput | RunStepUpdateInput) {
        return {
            id: data.id,
            contextSwitches: data.contextSwitches ?? undefined,
            timeElapsed: data.timeElapsed,
        }
    },
    shapeRelationshipCreate(userId: string, data: RunStepCreateInput): Prisma.run_routine_stepUncheckedCreateWithoutRunRoutineInput {
        return {
            ...this.shapeBase(userId, data),
            nodeId: data.nodeId,
            subroutineVersionId: data.subroutineVersionId,
            order: data.order,
            status: RunStepStatus.InProgress,
            step: data.step,
            title: data.title,
        }
    },
    shapeRelationshipUpdate(userId: string, data: RunStepUpdateInput): Prisma.run_routine_stepUncheckedUpdateWithoutRunRoutineInput {
        return {
            ...this.shapeBase(userId, data),
            status: data.status ?? undefined,
        }
    },
    shapeCreate(userId: string, data: RunStepCreateInput & { runId: string }): Prisma.run_routine_stepUpsertArgs['create'] {
        return {
            ...this.shapeRelationshipCreate(userId, data),
            runId: data.runId,
        }
    },
    shapeUpdate(userId: string, data: RunStepUpdateInput): Prisma.run_routine_stepUpsertArgs['update'] {
        return {
            ...this.shapeRelationshipUpdate(userId, data),
        }
    },
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'steps',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isTransferable: false,
            shape: { shapeCreate: this.shapeRelationshipCreate, shapeUpdate: this.shapeRelationshipUpdate },
            userId,
        });
    },
    async cud(params: CUDInput<RunStepCreateInput & { runId: string }, RunStepUpdateInput>): Promise<CUDResult<RunStep>> {
        return cudHelper({
            ...params,
            objectType: 'RunStep',
            prisma,
            yup: { yupCreate: stepsCreate, yupUpdate: stepsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const RunStepModel = ({
    prismaObject: (prisma: PrismaType) => prisma.run_routine_step,
    format: runStepFormatter(),
    mutate: runStepMutater,
    type: 'RunStep' as GraphQLModelType,
    verify: runStepVerifier(),
})