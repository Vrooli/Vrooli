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

// export const runInputSearcher = (): Searcher<RunSearchInput, RunSortBy, Prisma.run_routineOrderByWithRelationInput, Prisma.run_routineWhereInput> => ({
//     defaultSort: RunSortBy.DateUpdatedDesc,
//     sortMap:  {
//             DateStartedAsc: { timeStarted: 'asc' },
//             DateStartedDesc: { timeStarted: 'desc' },
//             DateCompletedAsc: { timeCompleted: 'asc' },
//             DateCompletedDesc: { timeCompleted: 'desc' },
//             DateCreatedAsc: { created_at: 'asc' },
//             DateCreatedDesc: { created_at: 'desc' },
//             DateUpdatedAsc: { updated_at: 'asc' },
//             DateUpdatedDesc: { updated_at: 'desc' },
//     },
//     getSearchStringQuery: (searchString, languages) => {
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
//     customQueries(input) {
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
    shapeRelationshipCreate(userId: string, data: RunInputCreateInput): Prisma.run_routine_inputUncheckedCreateWithoutRunRoutineInput {
        return {
            id: data.id,
            data: data.data,
            inputId: data.inputId,
        }
    },
    shapeRelationshipUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_routine_inputUncheckedUpdateWithoutRunRoutineInput {
        return {
            data: data.data
        }
    },
    shapeCreate(userId: string, data: RunInputCreateInput & { runId: string }): Prisma.run_routine_inputUpsertArgs['create'] {
        return {
            ...this.shapeRelationshipCreate(userId, data),
            runId: data.runId,
        }
    },
    shapeUpdate(userId: string, data: RunInputUpdateInput): Prisma.run_routine_inputUpsertArgs['update'] {
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
    prismaObject: (prisma: PrismaType) => prisma.run_routine_input,
    format: runInputFormatter(),
    mutate: runInputMutater,
    // search: runInputSearcher(),
    type: 'RunInput' as GraphQLModelType,
    verify: runInputVerifier(),
})