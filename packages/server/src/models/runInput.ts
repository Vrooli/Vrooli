import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { RunInput, RunInputCreateInput, RunInputUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { RunModel } from "./run";

const formatter = (): Formatter<RunInput, any> => ({
    relationshipMap: {
        __typename: 'RunInput',
        input: 'InputItem',
    },
})

const validator = (): Validator<
    RunInputCreateInput,
    RunInputUpdateInput,
    RunInput,
    Prisma.run_routine_inputGetPayload<{ select: { [K in keyof Required<Prisma.run_routine_inputSelect>]: true } }>,
    any,
    Prisma.run_routine_inputSelect,
    Prisma.run_routine_inputWhereInput
> => ({
    validateMap: {
        __typename: 'RunRoutine',
        input: 'InputItem',
        runRoutine: 'RunRoutine',
    },
    isTransferable: false,
    permissionsSelect: (...params) => ({
        runRoutine: { select: RunModel.validate.permissionsSelect(...params) }
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canView', async () => isPublic],
    ]),
    profanityFields: ['data'],
    ownerOrMemberWhere: (userId) => ({
        runRoutine: {
            routineVersion: {
                root: {
                    OR: [
                        { user: { id: userId } },
                        { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                    ]
                }
            }
        }
    }),
    owner: (data) => RunModel.validate.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunModel.validate.isPublic(data.runRoutine as any, languages),
})

// const searcher = (): Searcher<RunSearchInput, RunSortBy, Prisma.run_routineOrderByWithRelationInput, Prisma.run_routineWhereInput> => ({
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

/**
 * Handles mutations of run inputs
 */
const mutater = (): Mutater<
    RunInput,
    false,
    false,
    { graphql: RunInputCreateInput, db: Prisma.run_routine_inputCreateWithoutRunRoutineInput },
    { graphql: RunInputUpdateInput, db: Prisma.run_routine_inputUpdateWithoutRunRoutineInput }
> => ({
    shape: {
        relCreate: async ({ data }) => {
            return {
                id: data.id,
                data: data.data,
                input: { connect: { id: data.inputId } },
            }
        },
        relUpdate: async ({ data }) => {
            return {
                data: data.data
            }
        }
    },
    yup: { create: runInputsCreate, update: runInputsUpdate },
})

export const RunInputModel = ({
    delegate: (prisma: PrismaType) => prisma.run_routine_input,
    format: formatter(),
    mutate: mutater(),
    // search: searcher(),
    type: 'RunInput' as GraphQLModelType,
    validate: validator(),
})