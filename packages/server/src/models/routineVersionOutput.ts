import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic, Mutater } from "./types";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput } from '@shared/consts';

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionOutputCreateInput,
    GqlUpdate: RoutineVersionOutputUpdateInput,
    GqlModel: RoutineVersionOutput,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.routine_version_outputUpsertArgs['create'],
    PrismaUpdate: Prisma.routine_version_outputUpsertArgs['update'],
    PrismaModel: Prisma.routine_version_outputGetPayload<SelectWrap<Prisma.routine_version_outputSelect>>,
    PrismaSelect: Prisma.routine_version_outputSelect,
    PrismaWhere: Prisma.routine_version_outputWhereInput,
}

const __typename = 'RoutineVersionOutput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        routineVersion: 'RoutineVersion',
        standardVersion: 'StandardVersion',
    },
    prismaRelMap: {
        __typename,
        routineVersion: 'RoutineVersion',
        standardVersion: 'StandardVersion',
    },
    countFields: {},
})

// const mutater = (): Mutater<Model> => ({
//     shape: {
//         create: async ({ prisma, userData, data }) => {
//             return {
//                 id: data.id,
//                 name: data.name,
//                 standardVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
//                 translations: await translationRelationshipBuilder(prisma, userData, data, true),
//             }
//         },
//         update: async ({ prisma, userData, data }) => {
//             return {
//                 name: data.name,
//                 standardVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
//                 translations: await translationRelationshipBuilder(prisma, userData, data, false),
//             }
//         },
//     },
//     yup: { create: {} as any, update: {} as any },
// })

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        name: true,
        routineVersion: padSelect(RoutineModel.display.select),
    }),
    label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
})

export const RoutineVersionOutputModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
})