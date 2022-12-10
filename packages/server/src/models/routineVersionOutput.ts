import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { relBuilderHelper } from "../actions";
import { Displayer, Formatter, Mutater } from "./types";
import { translationRelationshipBuilder } from "../utils";
import { RoutineModel } from "./routine";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";
import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput } from "../endpoints/types";

const __typename = 'RoutineVersionOutput' as const;

const suppFields = [] as const;
const formatter = (): Formatter<RoutineVersionOutput, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        routineVersion: 'RoutineVersion',
        standardVersion: 'StandardVersion',
    },
})

const mutater = (): Mutater<
    RoutineVersionOutput,
    false,
    false,
    { graphql: RoutineVersionOutputCreateInput, db: Prisma.routine_version_outputCreateWithoutRoutineVersionInput },
    { graphql: RoutineVersionOutputUpdateInput, db: Prisma.routine_version_outputUpdateWithoutRoutineVersionInput }
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
    Prisma.routine_version_outputGetPayload<SelectWrap<Prisma.routine_version_outputSelect>>
> => ({
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