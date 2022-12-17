import { runInputsCreate, runInputsUpdate } from "@shared/validation";
import { RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputSearchInput, RunRoutineInputSortBy, RunRoutineInputUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater, Searcher, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionInputModel } from ".";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineInputCreateInput,
    GqlUpdate: RunRoutineInputUpdateInput,
    GqlModel: RunRoutineInput,
    GqlSearch: RunRoutineInputSearchInput,
    GqlSort: RunRoutineInputSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.run_routine_inputUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_inputUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_inputGetPayload<SelectWrap<Prisma.run_routine_inputSelect>>,
    PrismaSelect: Prisma.run_routine_inputSelect,
    PrismaWhere: Prisma.run_routine_inputWhereInput,
}

const __typename = 'RunRoutineInput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        input: 'RoutineVersionInput',
    },
    prismaRelMap: {
        __typename,
        input: 'RunRoutineInput',
        runRoutine: 'RunRoutine',
    }
})

const validator = (): Validator<Model> => ({
    isTransferable: false,
    maxObjects: 100000,
    permissionsSelect: (...params) => ({
        id: true,
        runRoutine: 'RunRoutine',
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ({
        canDelete: async () => isAdmin,
        canEdit: async () => isAdmin,
        canView: async () => isPublic,
    }),
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

const searcher = (): Searcher<Model> => ({
    defaultSort: RunRoutineInputSortBy.DateUpdatedDesc,
    sortBy: RunRoutineInputSortBy,
    searchFields: [
        'createdTimeFrame',
        'excludeIds',
        'routineIds',
        'standardIds',
        'updatedTimeFrame',
    ],
    searchStringQuery: () => ({ runRoutine: RunRoutineModel.search.searchStringQuery() }),
})

/**
 * Handles mutations of run inputs
 */
const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data }) => {
            return {
                // id: data.id,
                // data: data.data,
                // input: { connect: { id: data.inputId } },
            } as any;
        },
        update: async ({ data }) => {
            return {
                data: data.data
            }
        }
    },
    yup: { create: runInputsCreate, update: runInputsUpdate },
})

const displayer = (): Displayer<Model> => ({
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
    search: searcher(),
    validate: validator(),
})