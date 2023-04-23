import { routineVersionOutputValidation } from "@local/validation";
import { noNull, selPad, shapeHelper } from "../builders";
import { translationShapeHelper } from "../utils";
import { RoutineModel } from "./routine";
const __typename = "RoutineVersionOutput";
const suppFields = [];
export const RoutineVersionOutputModel = ({
    __typename,
    delegate: (prisma) => prisma.routine_version_output,
    display: {
        select: () => ({
            id: true,
            name: true,
            routineVersion: selPad(RoutineModel.display.select),
        }),
        label: (select, languages) => select.name ?? RoutineModel.display.label(select.routineVersion, languages),
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
//# sourceMappingURL=routineVersionOutput.js.map