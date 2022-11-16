import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { relationshipBuilderHelper } from "./builder";
import { RunInput, RunInputCreateInput, RunInputUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

export const runInputFormatter = (): FormatConverter<RunInput, any> => ({
    relationshipMap: {
        __typename: 'RunInput',
        input: 'InputItem',
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
export const runInputMutater = (prisma: PrismaType): Mutater<RunInput> => ({
    shapeRelationshipCreate(userId: string, data: RunInputCreateInput): Prisma.run_inputUncheckedCreateWithoutRunInput {
        return {
            id: data.id,
            data: data.data,
            inputId: data.inputId,
        }
    },
    shapeRelationshipUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_inputUncheckedUpdateWithoutRunInput {
        return {
            data: data.data
        }
    },
    shapeCreate(userId: string, data: RunInputCreateInput & { runId: string }): Prisma.run_inputUpsertArgs['create'] {
        return {
            ...this.shapeRelationshipCreate(userId, data),
            runId: data.runId,
        }
    },
    shapeUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_inputUpsertArgs['update'] {
        return {
            ...this.shapeRelationshipUpdate(userId, data),
        }
    },
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'inputs',
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
    async cud(params: CUDInput<RunInputCreateInput & { runId: string }, RunInputUpdateInput>): Promise<CUDResult<RunInput>> {
        return cudHelper({
            ...params,
            objectType: 'RunInput',
            prisma,
            yup: { yupCreate: runInputsCreate, yupUpdate: runInputsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
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