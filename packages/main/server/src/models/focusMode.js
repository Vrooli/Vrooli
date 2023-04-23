import { FocusModeSortBy, MaxObjects } from "@local/consts";
import { focusModeValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions, labelShapeHelper } from "../utils";
const __typename = "FocusMode";
const suppFields = [];
export const FocusModeModel = ({
    __typename,
    delegate: (prisma) => prisma.focus_mode,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            filters: "FocusModeFilter",
            labels: "Label",
            reminderList: "ReminderList",
            schedule: "Schedule",
        },
        prismaRelMap: {
            __typename,
            reminderList: "ReminderList",
            resourceList: "ResourceList",
            user: "User",
            labels: "Label",
            filters: "FocusModeFilter",
            schedule: "Schedule",
        },
        countFields: {},
        joinMap: { labels: "label" },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                user: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "filters", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "FocusModeFilter", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: false, objectType: "ReminderList", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "focusModes", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "FocusMode", relation: "labels", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                ...(await shapeHelper({ relation: "filters", relTypes: ["Create", "Delete"], isOneToOne: false, isRequired: false, objectType: "FocusModeFilter", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Disconnect", "Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ReminderList", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "focusMode", data, ...rest })),
                ...(await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "focusModes", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Disconnect", "Create"], parentType: "FocusMode", relation: "labels", data, ...rest })),
            }),
        },
        yup: focusModeValidation,
    },
    search: {
        defaultSort: FocusModeSortBy.NameAsc,
        sortBy: FocusModeSortBy,
        searchFields: {
            createdTimeFrame: true,
            scheduleStartTimeFrame: true,
            scheduleEndTimeFrame: true,
            recurrStartTimeFrame: true,
            recurrEndTimeFrame: true,
            labelsIds: true,
            timeZone: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "nameWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
});
//# sourceMappingURL=focusMode.js.map