import { Prisma } from "@prisma/client";
import { OutputItem, OutputItemCreateInput, OutputItemUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relBuilderHelper } from "./actions";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { translationRelationshipBuilder } from "./utils";

const formatter = (): Formatter<OutputItem, any> => ({
    relationshipMap: {
        __typename: 'OutputItem',
        standard: 'Standard',
    },
})

const mutater = (): Mutater<
    OutputItem,
    false,
    false,
    { graphql: OutputItemCreateInput, db: Prisma.routine_version_outputCreateWithoutRoutineVersionInput },
    { graphql: OutputItemUpdateInput, db: Prisma.routine_version_outputUpdateWithoutRoutineVersionInput }
> => ({
    shape: {
        relCreate: async ({ prisma, userData, data }) => {
            return {
                id: data.id,
                name: data.name,
                standard: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
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

export const OutputItemModel = ({
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    format: formatter(),
    mutate: mutater(),
    type: 'OutputItem' as GraphQLModelType,
})