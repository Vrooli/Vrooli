import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput, routineVersionInputValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { translationShapeHelper } from "../../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "RoutineVersionInput" as const;
export const RoutineVersionInputFormat: Formatter<ModelRoutineVersionInputLogic> = {
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
};
