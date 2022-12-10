import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionInputModel } from ".";

const __typename = 'RunRoutineInput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<RunRoutineInput, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        input: 'RoutineVersionInput',
    },
})

const validator = (): Validator<
    RunRoutineInputCreateInput,
    RunRoutineInputUpdateInput,
    Prisma.run_routine_inputGetPayload<SelectWrap<Prisma.run_routine_inputSelect>>,
    any,
    Prisma.run_routine_inputSelect,
    Prisma.run_routine_inputWhereInput,
    false,
    false
> => ({
    validateMap: {
        __typename: 'RunRoutine',
        input: 'RoutineVersionInput',
        runRoutine: 'RunRoutine',
    },
    isTransferable: false,
    maxObjects: 100000,
    permissionsSelect: (...params) => ({
        runRoutine: { select: RunRoutineModel.validate.permissionsSelect(...params) }
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canView', async () => isPublic],
    ]),
    profanityFields: ['data'],
    owner: (data) => RunRoutineModel.validate.owner(data.runRoutine as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RunRoutineModel.validate.isPublic(data.runRoutine as any, languages),
    visibility: {
        private: { runRoutine: { isPrivate: true } },
        public: { runRoutine: { isPrivate: false } },
        owner: (userId) => ({ runRoutine: RunRoutineModel.validate.visibility.owner(userId) }),
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
//                         translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } },
//                     }
//                 },
//                 { name: { ...insensitive } }
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
    RunRoutineInput,
    false,
    false,
    { graphql: RunRoutineInputCreateInput, db: Prisma.run_routine_inputCreateWithoutRunRoutineInput },
    { graphql: RunRoutineInputUpdateInput, db: Prisma.run_routine_inputUpdateWithoutRunRoutineInput }
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
    Prisma.run_routine_inputGetPayload<SelectWrap<Prisma.run_routine_inputSelect>>
> => ({
    select: () => ({
        id: true,
        input: padSelect(RoutineVersionInputModel.display.select),
        runRoutine: padSelect(RunRoutineModel.display.select),
    }),
    // Label combines runRoutine's label and input's label
    label: (select, languages) => {
        const runRoutineLabel = RunRoutineModel.display.label(select.runRoutine as any, languages);
        const inputLabel = RoutineVersionInputModel.display.label(select.input as any, languages);
        if (runRoutineLabel.length > 0) {
            return `${runRoutineLabel} - ${inputLabel}`;
        }
        return inputLabel;
    }
})

export const RunRoutineInputModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_input,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    // search: searcher(),
    validate: validator(),
})