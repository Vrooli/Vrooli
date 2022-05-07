import { CODE, runCreate, runUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { Count, LogType, Run, RunCancelInput, RunCompleteInput, RunCreateInput, RunSearchInput, RunSortBy, RunStatus, RunUpdateInput } from "../../schema/types";
import { PrismaType } from "../../types";
import { addSupplementalFields, CUDInput, CUDResult, FormatConverter, GraphQLModelType, InfoType, modelToGraphQL, Searcher, selectHelper, timeFrameToPrisma, toPartialSelect, ValidateMutationsInput } from "./base";
import _ from "lodash";
import { genErrorCode } from "../../logger";
import { Log } from "../../models/nosql";
import { TranslationModel } from "./translation";
import { StepModel } from "./step";

//==============================================================
/* #region Custom Components */
//==============================================================

export const runFormatter = (): FormatConverter<Run> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Run,
        'routine': GraphQLModelType.Routine,
        'steps': GraphQLModelType.RunStep,
        'user': GraphQLModelType.User,
    },
})

export const runSearcher = (): Searcher<RunSearchInput> => ({
    defaultSort: RunSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RunSortBy.DateStartedAsc]: { timeStarted: 'asc' },
            [RunSortBy.DateStartedDesc]: { timeStarted: 'desc' },
            [RunSortBy.DateCompletedAsc]: { timeCompleted: 'asc' },
            [RunSortBy.DateCompletedDesc]: { timeCompleted: 'desc' },
            [RunSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [RunSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [RunSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [RunSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                {
                    routine: {
                        OR: [
                            { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
                            { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
                        ]
                    }
                },
                { title: { ...insensitive } }
            ]
        })
    },
    customQueries(input: RunSearchInput): { [x: string]: any } {
        return {
            ...(input.routineId ? { routines: { some: { id: input.routineId } } } : {}),
            ...(input.completedTimeFrame ? timeFrameToPrisma('timeCompleted', input.completedTimeFrame) : {}),
            ...(input.startedTimeFrame ? timeFrameToPrisma('timeStarted', input.startedTimeFrame) : {}),
            ...(input.status ? { status: input.status } : {}),
        }
    },
})

/**
 * Handles run instances of routines
 */
export const runMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string, data: RunCreateInput): Promise<any> {
        // TODO - when scheduling added, don't assume that it is being started right away
        return {
            timeStarted: new Date(),
            routineId: data.routineId,
            status: RunStatus.InProgress,
            title: data.title,
            userId,
            version: data.version,
        }
    },
    async toDBShapeUpdate(userId: string, data: RunUpdateInput): Promise<any> {
        return {
            timeElapsed: data.timeElapsed ?? undefined,
            completedComplexity: data.completedComplexity ?? undefined,
            pickups: data.pickups ?? undefined,
            steps: await StepModel(prisma).relationshipBuilder(userId, data, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RunCreateInput, RunUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0174') });
        if (createMany) {
            createMany.forEach(input => runCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
        }
        if (updateMany) {
            for (const input of updateMany) {
                runUpdate.validateSync(input.data, { abortEarly: false });
                TranslationModel().profanityCheck(input.data);
            }
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
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<RunCreateInput, RunUpdateInput>): Promise<CUDResult<Run>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        if (!userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0175') });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId, input);
                // Create object
                const currCreated = await prisma.run.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
            // Log run starts 
            const logData: any[] = [];
            for (let i = 0; i < created.length; i++) {
                logData.push({
                    timestamp: Date.now(),
                    userId,
                    action: LogType.RoutineStartIncomplete,
                    object1Type: GraphQLModelType.Run,
                    object1Id: created[i].id,
                    object2Type: GraphQLModelType.Routine,
                    object2Id: createMany[i].routineId,
                })
            }
            Log.collection.insertMany(logData);
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.run.findFirst({
                    where: { ...input.where, userId }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Run not found.', { code: genErrorCode('0176') });
                // Update object
                const currUpdated = await prisma.run.update({
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
            deleted = await prisma.run.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
    /**
     * Deletes all runs for a user, except if they are in progress
     */
    async deleteAll(userId: string): Promise<Count> {
        return prisma.run.deleteMany({
            where: {
                AND: [
                    { userId },
                    { NOT: { status: RunStatus.InProgress } }
                ]
            }
        });
    },
    /**
     * Marks a run as completed
     */
    async complete(userId: string, input: RunCompleteInput, info: InfoType): Promise<Run> {
        // Convert info to partial
        const partial = toPartialSelect(info, runFormatter().relationshipMap);
        if (partial === undefined) throw new CustomError(CODE.ErrorUnknown, 'Invalid query.', { code: genErrorCode('0179') });
        // Find in database
        let object = await prisma.run.findFirst({
            where: {
                AND: [
                    { userId },
                    { id: input.id },
                ]
            }
        })
        if (!object) throw new CustomError(CODE.NotFound, 'Run not found.', { code: genErrorCode('0180') });
        // Update object
        const updated = await prisma.run.update({
            where: { id: input.id },
            data: {
                pickups: input.pickups ?? undefined,
                timeCompleted: new Date(),
                timeElapsed: input.timeElapsed ?? undefined,
            },
            ...selectHelper(partial)
        });
        // Convert to GraphQL
        let converted: any = modelToGraphQL(updated, partial);
        // Add supplemental fields
        converted = addSupplementalFields(prisma, userId, [converted], partial);
        // Log run completion
        Log.collection.insertOne({
            timestamp: Date.now(),
            userId,
            action: LogType.RoutineComplete,
            object1Type: GraphQLModelType.Run,
            object1Id: input.id,
            object2Type: GraphQLModelType.Routine,
            object2Id: object.routineId,
        });
        // Return converted object
        return converted as Run;
    },
    /**
     * Cancels a run
     */
    async cancel(userId: string, input: RunCancelInput, info: InfoType): Promise<Run> {
        // Convert info to partial
        const partial = toPartialSelect(info, runFormatter().relationshipMap);
        if (partial === undefined) throw new CustomError(CODE.ErrorUnknown, 'Invalid query.', { code: genErrorCode('0181') });
        // Find in database
        let object = await prisma.run.findFirst({
            where: {
                AND: [
                    { userId },
                    { id: input.id },
                ]
            }
        })
        if (!object) throw new CustomError(CODE.NotFound, 'Run not found.', { code: genErrorCode('0182') });
        // Update object
        const updated = await prisma.run.update({
            where: { id: input.id },
            data: {
                status: RunStatus.Cancelled,
            },
            ...selectHelper(partial)
        });
        // Convert to GraphQL
        let converted: any = modelToGraphQL(updated, partial);
        // Add supplemental fields
        converted = addSupplementalFields(prisma, userId, [converted], partial);
        // Log run cancellation
        Log.collection.insertOne({
            timestamp: Date.now(),
            userId,
            action: LogType.RoutineCancel,
            object1Type: GraphQLModelType.Run,
            object1Id: input.id,
            object2Type: GraphQLModelType.Routine,
            object2Id: object.routineId,
        });
        // Return converted object
        return converted as Run;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RunModel(prisma: PrismaType) {
    const prismaObject = prisma.run;
    const format = runFormatter();
    const search = runSearcher();
    const mutate = runMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...mutate,
    }
}