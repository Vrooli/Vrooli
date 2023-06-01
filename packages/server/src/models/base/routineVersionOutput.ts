import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput, routineVersionOutputValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { translationShapeHelper } from "../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = "RoutineVersionOutput" as const;
const suppFields = [] as const;
export const RoutineVersionOutputModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionOutputCreateInput,
    GqlUpdate: RoutineVersionOutputUpdateInput,
    GqlModel: RoutineVersionOutput,
    GqlPermission: object,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.routine_version_outputUpsertArgs["create"],
    PrismaUpdate: Prisma.routine_version_outputUpsertArgs["update"],
    PrismaModel: Prisma.routine_version_outputGetPayload<SelectWrap<Prisma.routine_version_outputSelect>>,
    PrismaSelect: Prisma.routine_version_outputSelect,
    PrismaWhere: Prisma.routine_version_outputWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_output,
    display: {
        label: {
            select: () => ({
                id: true,
                name: true,
                routineVersion: selPad(RoutineModel.display.label.select),
            }),
            get: (select, languages) => select.name ?? RoutineModel.display.label.get(select.routineVersion as any, languages),
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            routineVersion: "RoutineVersion",
            standardVersion: "StandardVersion",
        },
        prismaRelMap: {
            __typename,
            routineVersion: "RoutineVersion",
            standardVersion: "StandardVersion",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "outputs", data, ...rest })),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Create", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: routineVersionOutputValidation,
    },
});
