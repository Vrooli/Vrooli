import { routineVersionOutputValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../../builders";
import { translationShapeHelper } from "../../utils";
import { RoutineVersionOutputFormat } from "../format/routineVersionOutput";
import { ModelLogic } from "../types";
import { RoutineModel } from "./routine";
import { RoutineVersionOutputModelLogic } from "./types";

const __typename = "RoutineVersionOutput" as const;
const suppFields = [] as const;
export const RoutineVersionOutputModel: ModelLogic<RoutineVersionOutputModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.routine_version_output,
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
    format: RoutineVersionOutputFormat,
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
