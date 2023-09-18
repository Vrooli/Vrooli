import { routineVersionInputValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, translationShapeHelper } from "../../utils";
import { RoutineVersionInputFormat } from "../formats";
import { ModelLogic } from "../types";
import { RoutineVersionModel } from "./routineVersion";
import { RoutineVersionInputModelLogic, RoutineVersionModelLogic } from "./types";

const __typename = "RoutineVersionInput" as const;
const suppFields = [] as const;
export const RoutineVersionInputModel: ModelLogic<RoutineVersionInputModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.routine_version_input,
    display: {
        label: {
            select: () => ({
                id: true,
                name: true,
                routineVersion: { select: RoutineVersionModel.display.label.select() },
            }),
            get: (select, languages) => select.name ?? RoutineVersionModel.display.label.get(select.routineVersion as RoutineVersionModelLogic["PrismaModel"], languages),
        },
    },
    format: RoutineVersionInputFormat,
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
    search: undefined,
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => RoutineVersionModel.validate.isPublic(data.routineVersion as RoutineVersionModelLogic["PrismaModel"], languages),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => RoutineVersionModel.validate.owner(data.routineVersion as RoutineVersionModelLogic["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ routineVersion: RoutineVersionModel.validate.visibility.owner(userId) }),
        },
    },
});
