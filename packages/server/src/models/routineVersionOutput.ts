import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput } from '@shared/consts';

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
//     yup: routineVersionOutputValidation,
// })

const __typename = 'RoutineVersionOutput' as const;
const suppFields = [] as const;
export const RoutineVersionOutputModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    display: {
        select: () => ({
            id: true,
            name: true,
            routineVersion: padSelect(RoutineModel.display.select),
        }),
        label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
    },
    format: {
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
    },
    mutate: {} as any,//mutater(),
})