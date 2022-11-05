import { runsCreate, runsUpdate } from "@shared/validation";
import { CODE, RunSortBy } from "@shared/consts";
import { addSupplementalFields, modelToGraphQL, selectHelper, timeFrameToPrisma, toPartialGraphQLInfo, getSearchStringQueryHelper, combineQueries, onlyValidIds } from "./builder";
import { RunStepModel } from "./runStep";
import { run, RunStatus } from "@prisma/client";
import { RunInputModel } from "./runInput";
import { organizationQuerier } from "./organization";
import { routinePermissioner } from "./routine";
import { CustomError, genErrorCode, logger, LogLevel, Trigger } from "../events";
import { Run, RunSearchInput, RunCreateInput, RunUpdateInput, RunPermission, Count, LogType, RunCompleteInput, RunCancelInput } from "../schema/types";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { FormatConverter, Searcher, Permissioner, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType, GraphQLInfo } from "./types";

//==============================================================
/* #region Custom Components */
//==============================================================

export const runFormatter = (): FormatConverter<Run, any> => ({
    relationshipMap: {
        '__typename': 'Run',
        'routine': 'Routine',
        'steps': 'RunStep',
        'inputs': 'RunInput',
        'user': 'User',
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
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    {
                        routine: {
                            translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } },
                        }
                    },
                    {
                        routine: {
                            translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } },
                        }
                    },
                    { title: { ...insensitive } }
                ]
            })
        })
    },
    customQueries(input: RunSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            (input.completedTimeFrame !== undefined ? timeFrameToPrisma('timeCompleted', input.completedTimeFrame) : {}),
            (input.startedTimeFrame !== undefined ? timeFrameToPrisma('timeStarted', input.startedTimeFrame) : {}),
            (input.status !== undefined ? { status: input.status } : {}),
        ])
    },
})

export const runVerifier = () => ({
    profanityCheck(data: (RunCreateInput | RunUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.title));
    },
})

export const runPermissioner = (): Permissioner<RunPermission, RunSearchInput> => ({
    async get({
        objects,
        permissions,
        prisma,
        userId,
    }) {
        // Initialize result with default permissions
        const result: (RunPermission & { id?: string })[] = objects.map((o) => ({
            id: o.id,
            canDelete: false, // own || (own associated routine && !isPrivate)
            canEdit: false, // own
            canView: false, // own || (own associated routine && !isPrivate)
        }));
        // Check ownership
        if (userId) {
            // Query for objects with matching userId
            const owned = await prisma.run.findMany({
                where: {
                    id: { in: onlyValidIds(objects.map(o => o.id)) },
                    userId,
                },
                select: { id: true },
            })
            // Set permissions for owned objects
            owned.forEach((o) => {
                const index = objects.findIndex((r) => r.id === o.id);
                result[index] = {
                    ...result[index],
                    canDelete: true,
                    canEdit: true,
                    canView: true,
                }
            });
        }
        // Query all runs marked as public, where you own the associated routine
        const all = await prisma.run.findMany({
            where: {
                id: { in: onlyValidIds(objects.map(o => o.id)) },
                isPrivate: false,
                AND: [
                    { routineId: null },
                    routinePermissioner().ownershipQuery(userId ?? ''),
                ]
            },
            select: { id: true },
        })
        // Set permissions for found objects
        all.forEach((o) => {
            const index = objects.findIndex((r) => r.id === o.id);
            result[index] = {
                ...result[index],
                canDelete: true,
                canView: true,
            }
        });
        // Return result with IDs removed
        result.forEach((r) => delete r.id);
        return result as RunPermission[];
    },
    async canSearch({
        input,
        prisma,
        userId
    }) {
        //TODO
        return 'full';
    },
    ownershipQuery: (userId) => ({
        routine: {
            OR: [
                organizationQuerier().hasRoleInOrganizationQuery(userId),
                { user: { id: userId } }
            ]
        }
    })
})

/**
 * Handles run instances of routines
 */
export const runMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string, data: RunCreateInput): Promise<any> {
        // TODO - when scheduling added, don't assume that it is being started right away
        return {
            id: data.id,
            timeStarted: new Date(),
            routineId: data.routineId,
            status: RunStatus.InProgress,
            steps: await RunStepModel.mutate(prisma).relationshipBuilder(userId, data, true, 'step'),
            title: data.title,
            userId,
            version: data.version,
        }
    },
    async toDBShapeUpdate(userId: string, updateData: RunUpdateInput, existingData: Run): Promise<any> {
        return {
            timeElapsed: (existingData.timeElapsed ?? 0) + (updateData.timeElapsed ?? 0),
            completedComplexity: (existingData.completedComplexity ?? 0) + (updateData.completedComplexity ?? 0),
            contextSwitches: (existingData.contextSwitches ?? 0) + (updateData.contextSwitches ?? 0),
            steps: await RunStepModel.mutate(prisma).relationshipBuilder(userId, updateData, false),
            inputs: await RunInputModel.mutate(prisma).relationshipBuilder(userId, updateData, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RunCreateInput, RunUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0174') });
        if (createMany) {
            runsCreate.validateSync(createMany, { abortEarly: false });
            runVerifier().profanityCheck(createMany);
        }
        if (updateMany) {
            runsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            runVerifier().profanityCheck(updateMany.map(u => u.data));
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
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<RunCreateInput, RunUpdateInput>): Promise<CUDResult<Run>> {
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
                const currCreated = await prisma.run.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
            // Handle trigger
            for (const c of created) {
                await Trigger(prisma).runStart(c.id, userId);
            }
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
                const data = await this.toDBShapeUpdate(userId, input.data, object as any)
                const currUpdated = await prisma.run.update({
                    where: input.where,
                    data,
                    ...selectHelper(partialInfo)
                });
                // if (data.hasOwnProperty('wasSuccessful')) {
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
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
     * Marks a run as completed. Run does not have to exist, since this can be called on simple routines 
     * via the "Mark as Complete" button. We could create a new run every time a simple routine is viewed 
     * to get around this, but I'm not sure if that would be a good idea. Most of the time, I imagine users
     * will just be looking at the routine instead of using it.
     */
    async complete(userId: string, input: RunCompleteInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, runFormatter().relationshipMap);
        if (partial === undefined) throw new CustomError(CODE.ErrorUnknown, 'Invalid query.', { code: genErrorCode('0179') });
        let run: run | null;
        // Check if run is being created or updated
        if (input.exists) {
            // Find in database
            run = await prisma.run.findFirst({
                where: {
                    AND: [
                        { userId },
                        { id: input.id },
                    ]
                }
            })
            if (!run) throw new CustomError(CODE.NotFound, 'Run not found.', { code: genErrorCode('0180') });
            const { timeElapsed, contextSwitches, completedComplexity } = run;
            // Update object
            run = await prisma.run.update({
                where: { id: input.id },
                data: {
                    completedComplexity: completedComplexity + (input.completedComplexity ?? 0),
                    contextSwitches: contextSwitches + (input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0),
                    status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
                    timeCompleted: new Date(),
                    timeElapsed: (timeElapsed ?? 0) + (input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0),
                    steps: {
                        create: input.finalStepCreate ? {
                            order: input.finalStepCreate.order ?? 1,
                            title: input.finalStepCreate.title ?? '',
                            contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepCreate.timeElapsed,
                            status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
                        } as any : undefined,
                        update: input.finalStepUpdate ? {
                            id: input.finalStepUpdate.id,
                            contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepUpdate.timeElapsed,
                            status: input.finalStepUpdate.status ?? (input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed),
                        } as any : undefined,
                    }
                    //TODO
                    // inputs: {
                    //     create: input.finalInputCreate ? {
                    // }
                },
                ...selectHelper(partial)
            });
        } else {
            // Create new run
            run = await prisma.run.create({
                data: {
                    completedComplexity: input.completedComplexity ?? 0,
                    timeStarted: new Date(),
                    timeCompleted: new Date(),
                    timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
                    contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
                    routineId: input.id,
                    status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                    title: input.title,
                    userId,
                    version: input.version,
                    steps: {
                        create: input.finalStepCreate ? {
                            order: input.finalStepCreate.order ?? 1,
                            title: input.finalStepCreate.title ?? '',
                            contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepCreate.timeElapsed,
                            status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                        } as any : input.finalStepUpdate ? {
                            id: input.finalStepUpdate.id,
                            contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepUpdate.timeElapsed,
                            status: input.finalStepUpdate?.status ?? (input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed),
                        } : undefined,
                    }
                    //TODO inputs
                },
                ...selectHelper(partial)
            });
        }
        // Convert to GraphQL
        let converted: any = modelToGraphQL(run, partial);
        // Add supplemental fields
        converted = (await addSupplementalFields(prisma, userId, [converted], partial))[0];
        // Handle trigger
        await Trigger(prisma).runComplete(input.id, userId, input.wasSuccessful ?? false);
        // Return converted object
        return converted as Run;
    },
    /**
     * Cancels a run
     */
    async cancel(userId: string, input: RunCancelInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, runFormatter().relationshipMap);
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
        converted = (await addSupplementalFields(prisma, userId, [converted], partial))[0];
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

export const RunModel = ({
    prismaObject: (prisma: PrismaType) => prisma.run,
    format: runFormatter(),
    mutate: runMutater,
    permissions: runPermissioner,
    search: runSearcher(),
    type: 'Run' as GraphQLModelType,
    verify: runVerifier(),
})