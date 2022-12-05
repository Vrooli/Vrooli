import { Prisma } from "@prisma/client";
import { OutputItem, OutputItemCreateInput, OutputItemUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";

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
                standardVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'standard', objectType: 'Standard', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
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
    Prisma.routine_version_outputSelect,
    Prisma.routine_version_outputGetPayload<{ select: { [K in keyof Required<Prisma.routine_version_outputSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        name: true,
        routineVersion: padSelect(RoutineModel.display.select),
    }),
    label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
})

export const OutputItemModel = ({
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'OutputItem' as GraphQLModelType,
})