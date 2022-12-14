import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Displayer, Formatter, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput } from "../endpoints/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionOutputCreateInput,
    GqlUpdate: RoutineVersionOutputUpdateInput,
    GqlModel: RoutineVersionOutput,
    GqlPermission: any,
    PrismaCreate: Prisma.routine_version_outputUpsertArgs['create'],
    PrismaUpdate: Prisma.routine_version_outputUpsertArgs['update'],
    PrismaModel: Prisma.routine_version_outputGetPayload<SelectWrap<Prisma.routine_version_outputSelect>>,
    PrismaSelect: Prisma.routine_version_outputSelect,
    PrismaWhere: Prisma.routine_version_outputWhereInput,
}

const __typename = 'RoutineVersionOutput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        routineVersion: 'RoutineVersion',
        standardVersion: 'StandardVersion',
    },
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ prisma, userData, data }) => {
            return {
                id: data.id,
                name: data.name,
                standardVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
            }
        },
        update: async ({ prisma, userData, data }) => {
            return {
                name: data.name,
                standardVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: { create: {} as any, update: {} as any },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        name: true,
        routineVersion: padSelect(RoutineModel.display.select),
    }),
    label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
})

export const RoutineVersionOutputModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})