import { Prisma } from "@prisma/client";
import { InputItem, InputItemCreateInput, InputItemUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";

const formatter = (): Formatter<InputItem, any> => ({
    relationshipMap: {
        __typename: 'InputItem',
        standard: 'Standard',
    },
})

const mutater = (): Mutater<
    InputItem,
    false,
    false,
    { graphql: InputItemCreateInput, db: Prisma.routine_version_inputCreateWithoutRoutineVersionInput },
    { graphql: InputItemUpdateInput, db: Prisma.routine_version_inputUpdateWithoutRoutineVersionInput }
> => ({
    shape: {
        relCreate: async ({ prisma, userData, data }) => {
            return {
                id: data.id,
                name: data.name,
                standardVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true)
            }
        },
        relUpdate: async ({ prisma, userData, data }) => {
            return {
                name: data.name,
                standardVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: {},
})

const displayer = (): Displayer<
    Prisma.routine_version_inputSelect,
    Prisma.routine_version_inputGetPayload<{ select: { [K in keyof Required<Prisma.routine_version_inputSelect>]: true } }>
> => ({
    select: {
        id: true,
        name: true,
        routineVersion: padSelect(RoutineModel.display.select),
    },
    label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
})

export const InputItemModel = ({
    delegate: (prisma: PrismaType) => prisma.routine_version_input,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'InputItem' as GraphQLModelType,
})