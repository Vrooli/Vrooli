import { FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeSortBy, FocusModeUpdateInput, focusModeValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, labelShapeHelper } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "FocusMode" as const;
export const FocusModeFormat: Formatter<ModelFocusModeLogic> = {
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
        searchFields: {
            createdTimeFrame: true,
            scheduleStartTimeFrame: true,
            scheduleEndTimeFrame: true,
            labelsIds: true,
            timeZone: true,
            updatedTimeFrame: true,
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "nameWrapped",
            ],
        owner: (data) => ({
            User: data.user,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
};
