import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput, routineVersionInputValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { translationShapeHelper } from "../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = "RoutineVersionInput" as const;
const suppFields = [] as const;
export const RoutineVersionInputModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionInputCreateInput,
    GqlUpdate: RoutineVersionInputUpdateInput,
    GqlModel: RoutineVersionInput,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.routine_version_inputUpsertArgs["create"],
    PrismaUpdate: Prisma.routine_version_inputUpsertArgs["update"],
    PrismaModel: Prisma.routine_version_inputGetPayload<SelectWrap<Prisma.routine_version_inputSelect>>,
    PrismaSelect: Prisma.routine_version_inputSelect,
    PrismaWhere: Prisma.routine_version_inputWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_input,
    display: {
        select: () => ({
            id: true,
            name: true,
            routineVersion: selPad(RoutineModel.display.select),
        }),
        label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion as any, languages),
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
            runInputs: "RunRoutineInput",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                isRequired: noNull(data.isRequired),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "inputs", data, ...rest })),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "routineVersionInputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                isRequired: noNull(data.isRequired),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "routineVersionInputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Create", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: routineVersionInputValidation,
    },
});
