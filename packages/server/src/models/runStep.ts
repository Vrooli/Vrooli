import { stepsCreate, stepsUpdate } from "@shared/validation";
import { RunStepStatus } from "@shared/consts";
import { relationshipToPrisma, RelationshipTypes } from "./builder";
import { RunStep, RunStepCreateInput, RunStepUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

export const runStepFormatter = (): FormatConverter<RunStep, any> => ({
    relationshipMap: {
        '__typename': 'RunStep',
        'run': 'Run',
        'node': 'Node',
        'subroutine': 'Routine',
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
export const runStepMutater = (prisma: PrismaType) => ({
    shapeBase(userId: string, data: RunStepCreateInput | RunStepUpdateInput) {
        return {
            id: data.id,
            contextSwitches: data.contextSwitches ?? undefined,
            timeElapsed: data.timeElapsed,
        }
    },
    shapeRelationshipCreate(userId: string, data: RunStepCreateInput): Prisma.run_stepUncheckedCreateWithoutRunInput {
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
    shapeRelationshipUpdate(userId: string, data: RunStepUpdateInput): Prisma.run_stepUncheckedUpdateWithoutRunInput {
        return {
            ...this.shapeBase(userId, data),
            status: data.status ?? undefined,
        }
    },
    shapeCreate(userId: string, data: RunStepCreateInput & { runId: string }): Prisma.run_stepUpsertArgs['create'] {
        return {
            ...this.shapeRelationshipCreate(userId, data),
            runId: data.runId,
        }
    },
    shapeUpdate(userId: string, data: RunStepUpdateInput): Prisma.run_stepUpsertArgs['update'] {
        return {
            ...this.shapeRelationshipUpdate(userId, data),
        }
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'steps',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        // Further shape the input
        if (createMany) {
            let result: { [x: string]: any }[] = [];
            for (const data of createMany) {
                result.push(await this.shapeRelationshipCreate(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.shapeRelationshipUpdate(userId, data.data as any),
                })
            }
            updateMany = result;
        }
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    /**
     * Performs adds, updates, and deletes of steps. First validates that every action is allowed.
     */
    async cud(params: CUDInput<RunStepCreateInput & { runId: string }, RunStepUpdateInput>): Promise<CUDResult<RunStep>> {
        return cudHelper({
            ...params,
            objectType: 'RunStep',
            prisma,
            prismaObject: (p) => p.run_step,
            yup: { yupCreate: stepsCreate, yupUpdate: stepsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const RunStepModel = ({
    prismaObject: (prisma: PrismaType) => prisma.run_step,
    format: runStepFormatter(),
    mutate: runStepMutater,
    type: 'RunStep' as GraphQLModelType,
    verify: runStepVerifier(),
})