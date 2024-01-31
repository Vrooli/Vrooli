import { FocusModeSortBy, focusModeValidation, MaxObjects } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils";
import { labelShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { FocusModeFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { FocusModeModelLogic } from "./types";

const __typename = "FocusMode" as const;
export const FocusModeModel: FocusModeModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.focus_mode,
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: FocusModeFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                user: { connect: { id: rest.userData.id } },
                filters: await shapeHelper({ relation: "filters", relTypes: ["Create"], isOneToOne: false, objectType: "FocusModeFilter", parentRelationshipName: "focusMode", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "FocusMode", data, ...rest }),
                reminderList: await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "ReminderList", parentRelationshipName: "focusMode", data, ...rest }),
                resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "focusMode", data, ...rest }),
                schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "focusModes", data, ...rest }),

            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                filters: await shapeHelper({ relation: "filters", relTypes: ["Create", "Delete"], isOneToOne: false, objectType: "FocusModeFilter", parentRelationshipName: "focusMode", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Disconnect", "Create"], parentType: "FocusMode", data, ...rest }),
                reminderList: await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Disconnect", "Create", "Update"], isOneToOne: true, objectType: "ReminderList", parentRelationshipName: "focusMode", data, ...rest }),
                resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "focusMode", data, ...rest }),
                schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "focusModes", data, ...rest }),
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
            labelsIds: true,
            timeZone: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["descriptionWrapped", "nameWrapped"] }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
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
    }),
});
