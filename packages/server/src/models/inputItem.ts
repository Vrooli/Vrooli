import { Prisma } from "@prisma/client";
import { InputItem, InputItemCreateInput, InputItemUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";

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
                standard: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true)
            }
        },
        relUpdate: async ({ prisma, userData, data }) => {
            return {
                name: data.name,
                standard: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: {},
})

export const InputItemModel = ({
    delegate: (prisma: PrismaType) => prisma.routine_version_input,
    format: formatter(),
    mutate: mutater(),
    type: 'InputItem' as GraphQLModelType,
})