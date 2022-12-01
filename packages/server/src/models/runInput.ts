import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { RunInput, RunInputCreateInput, RunInputUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { RunModel } from "./run";
import { padSelect } from "../builders";
import { InputItemModel } from "./inputItem";

const formatter = (): Formatter<RunInput, any> => ({
    relationshipMap: {
        __typename: 'RunInput',
        input: 'InputItem',
    },
})

const validator = (): Validator<
    RunInputCreateInput,
    RunInputUpdateInput,
    Prisma.run_routine_inputGetPayload<{ select: { [K in keyof Required<Prisma.run_routine_inputSelect>]: true } }>,
    any,
    Prisma.run_routine_inputSelect,
    Prisma.run_routine_inputWhereInput,
    false,
    false
> => ({
    validateMap: {
        __typename: 'RunRoutine',
        input: 'InputItem',
        runRoutine: 'RunRoutine',
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
    profanityFields: ['data'],
    owner: (data) => RunModel.validate.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunModel.validate.isPublic(data.runRoutine as any, languages),
    visibility: {
        private: { runRoutine: { isPrivate: true } },
        public: { runRoutine: { isPrivate: false } },
        owner: (userId) => ({ runRoutine: RunModel.validate.visibility.owner(userId) }),
    },
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

const displayer = (): Displayer<
    Prisma.run_routine_inputSelect,
    Prisma.run_routine_inputGetPayload<{ select: { [K in keyof Required<Prisma.run_routine_inputSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        input: padSelect(InputItemModel.display.select),
        runRoutine: padSelect(RunModel.display.select),
    }),
    // Label combines runRoutine's label and input's label
    label: (select, languages) => {
        const runRoutineLabel = RunModel.display.label(select.runRoutine as any, languages);
        const inputLabel = InputItemModel.display.label(select.input as any, languages);
        if (runRoutineLabel.length > 0) {
            return `${runRoutineLabel} - ${inputLabel}`;
        }
        return inputLabel;
    }
})

export const RunInputModel = ({
    delegate: (prisma: PrismaType) => prisma.run_routine_input,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    // search: searcher(),
    type: 'RunInput' as GraphQLModelType,
    validate: validator(),
})