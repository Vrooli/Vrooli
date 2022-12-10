import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Displayer, Formatter, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput } from "../endpoints/types";

const __typename = 'RoutineVersionInput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<RoutineVersionInput, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        routineVersion: 'RoutineVersion',
        standardVersion: 'StandardVersion',
    },
})

const mutater = (): Mutater<
    RoutineVersionInput,
    false,
    false,
    { graphql: RoutineVersionInputCreateInput, db: Prisma.routine_version_inputCreateWithoutRoutineVersionInput },
    { graphql: RoutineVersionInputUpdateInput, db: Prisma.routine_version_inputUpdateWithoutRoutineVersionInput }
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
    Prisma.routine_version_inputGetPayload<SelectWrap<Prisma.routine_version_inputSelect>>
> => ({
    select: () => ({
        id: true,
        name: true,
        routineVersion: padSelect(RoutineModel.display.select),
    }),
    label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
})

export const RoutineVersionInputModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_input,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})