import { CODE, stepsCreate, stepsUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { Count, RunStep, RunStepCreateInput, RunStepStatus, RunStepUpdateInput} from "../../schema/types";
import { PrismaType } from "../../types";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper, ValidateMutationsInput } from "./base";
import _ from "lodash";
import { genErrorCode } from "../../logger";
import { validateProfanity } from "../../utils/censor";

//==============================================================
/* #region Custom Components */
//==============================================================

export const stepFormatter = (): FormatConverter<RunStep> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.RunStep,
        'run': GraphQLModelType.Run,
        'node': GraphQLModelType.Node,
    },
})

export const stepVerifier = () => ({
    profanityCheck(data: (RunStepCreateInput | RunStepUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.title));
    },
})

/**
 * Handles mutations of run steps
 */
 export const stepMutater = (prisma: PrismaType, verifier: ReturnType<typeof stepVerifier>) => ({
    async toDBShapeAdd(userId: string, data: RunStepCreateInput): Promise<any> {
        return {
            nodeId: data.nodeId,
            order: data.order,
            status: RunStepStatus.InProgress,
            step: data.step,
            title: data.title,
        }
    },
    async toDBShapeUpdate(userId: string, data: RunStepUpdateInput): Promise<any> {
        return {
            pickups: data.pickups ?? undefined,
            status: data.status ?? undefined,
            timeElapsed: data.timeElapsed ?? undefined,
        }
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'steps',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by nodes (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        // Further shape the input
        if (createMany) {
            let result = [];
            for (const data of createMany) {
                result.push(await this.toDBShapeAdd(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.toDBShapeUpdate(userId, data.data as any),
                })
            }
            updateMany = result;
        }
        // Validate input, with routine ID added to each update node
        await this.validateMutations({
            userId,
            createMany: createMany as RunStepCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: RunStepUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RunStepCreateInput, RunStepUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId) 
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0176') });
        if (createMany) {
            stepsCreate.validateSync(createMany, { abortEarly: false });
            verifier.profanityCheck(createMany);
        }
        if (updateMany) {
            stepsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            verifier.profanityCheck(updateMany.map(u => u.data));
            // Check that user owns each run
            //TODO
        }
        if (deleteMany) {
            // Check that user owns each run
            //TODO
        }
    },
    /**
     * Performs adds, updates, and deletes of runs. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<RunStepCreateInput, RunStepUpdateInput>): Promise<CUDResult<RunStep>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        if (!userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0177') });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId, input);
                // Create object
                const currCreated = await prisma.run_step.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.run_step.findFirst({
                    where: { ...input.where, run: { user: { id: userId } } },
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Step not found.', { code: genErrorCode('0176') });
                // Update object
                const currUpdated = await prisma.run_step.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId, input.data),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.run_step.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StepModel(prisma: PrismaType) {
    const prismaObject = prisma.run_step;
    const format = stepFormatter();
    const verify = stepVerifier();
    const mutate = stepMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        ...format,
        ...verify,
        ...mutate,
    }
}