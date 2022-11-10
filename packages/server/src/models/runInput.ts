import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { CODE } from "@shared/consts";
import { modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper } from "./builder";
import { CustomError, genErrorCode } from "../events";
import { RunInput, RunInputCreateInput, RunInputUpdateInput, Count } from "../schema/types";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { FormatConverter, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";

export const runInputFormatter = (): FormatConverter<RunInput, any> => ({
    relationshipMap: {
        '__typename': 'RunInput',
        'input': 'InputItem',
    },
})

// export const runInputSearcher = (): Searcher<RunSearchInput> => ({
//     defaultSort: RunSortBy.DateUpdatedDesc,
//     getSortQuery: (sortBy: string): any => {
//         return {
//             [RunSortBy.DateStartedAsc]: { timeStarted: 'asc' },
//             [RunSortBy.DateStartedDesc]: { timeStarted: 'desc' },
//             [RunSortBy.DateCompletedAsc]: { timeCompleted: 'asc' },
//             [RunSortBy.DateCompletedDesc]: { timeCompleted: 'desc' },
//             [RunSortBy.DateCreatedAsc]: { created_at: 'asc' },
//             [RunSortBy.DateCreatedDesc]: { created_at: 'desc' },
//             [RunSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
//             [RunSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
//         }[sortBy]
//     },
//     getSearchStringQuery: (searchString: string, languages?: string[]): any => {
//         const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
//         return ({
//             OR: [
//                 {
//                     routine: {
//                         translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } },
//                     }
//                 },
//                 {
//                     routine: {
//                         translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } },
//                     }
//                 },
//                 { title: { ...insensitive } }
//             ]
//         })
//     },
//     customQueries(input: RunSearchInput): { [x: string]: any } {
//         return combineQueries([
//             (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
//             (input.completedTimeFrame !== undefined ? timeFrameToPrisma('timeCompleted', input.completedTimeFrame) : {}),
//             (input.startedTimeFrame !== undefined ? timeFrameToPrisma('timeStarted', input.startedTimeFrame) : {}),
//             (input.status !== undefined ? { status: input.status } : {}),
//         ])
//     },
// })

export const runInputVerifier = () => ({
    profanityCheck(data: (RunInputCreateInput | RunInputUpdateInput)[]): void {
        validateProfanity(data.map((d) => d.data));
    },
})

/**
 * Handles mutations of run inputs
 */
export const runInputMutater = (prisma: PrismaType) => ({
    toDBRelationshipCreate(userId: string, data: RunInputCreateInput): Prisma.run_inputUncheckedCreateWithoutRunInput {
        return {
            id: data.id,
            data: data.data,
            inputId: data.inputId,
        }
    },
    toDBRelationshipUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_inputUncheckedUpdateWithoutRunInput {
        return {
            data: data.data
        }
    },
    toDBCreate(userId: string, data: RunInputCreateInput & { runId: string }): Prisma.run_inputUpsertArgs['create'] {
        return {
            ...this.toDBRelationshipCreate(userId, data),
            runId: data.runId,
        }
    },
    toDBUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_inputUpsertArgs['update'] {
        return {
            ...this.toDBRelationshipUpdate(userId, data),
        }
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'inputs',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        // Further shape the input
        if (createMany) {
            let result: { [x: string]: any }[] = [];
            for (const data of createMany) {
                result.push(await this.toDBRelationshipCreate(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result: { where: { [x: string]: string }, data: { [x: string]: any } }[] = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.toDBRelationshipUpdate(userId, data.data as any),
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
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RunInputCreateInput, RunInputUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (createMany) {
            runInputsCreate.validateSync(createMany, { abortEarly: false });
            runInputVerifier().profanityCheck(createMany);
        }
        if (updateMany) {
            runInputsUpdate.validateSync(updateMany.map(u => ({ ...u.data, id: u.where.id })), { abortEarly: false });
            runInputVerifier().profanityCheck(updateMany.map(u => u.data));
            // Check that user owns each input
            //TODO
        }
        if (deleteMany) {
            // Check that user owns each input
            //TODO
        }
    },
    /**
     * Performs adds, updates, and deletes of inputs. First validates that every action is allowed.
     */
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<RunInputCreateInput & { runId: string }, RunInputUpdateInput>): Promise<CUDResult<RunInput>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBCreate(userId, input);
                // Create object
                const currCreated = await prisma.run_input.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.run_input.findFirst({
                    where: { ...input.where, run: { user: { id: userId } } },
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Input not found.', { code: genErrorCode('0250') });
                // Update object
                const currUpdated = await prisma.run_input.update({
                    where: input.where,
                    data: await this.toDBUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.run_input.deleteMany({
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

export const RunInputModel = ({
    prismaObject: (prisma: PrismaType) => prisma.run_input,
    format: runInputFormatter(),
    mutate: runInputMutater,
    // search: runInputSearcher(),
    type: 'RunInput' as GraphQLModelType,
    verify: runInputVerifier(),
})