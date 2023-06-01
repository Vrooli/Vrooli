import { routineVersionInputValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { translationShapeHelper } from "../utils";
import { RoutineModel } from "./routine";
import { ModelLogic, RoutineVersionInputModelLogic } from "./types";

const __typename = "RoutineVersionInput" as const;
const suppFields = [] as const;
export const RoutineVersionInputModel: ModelLogic<RoutineVersionInputModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version_input,
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
