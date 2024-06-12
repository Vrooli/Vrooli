import { LabelSortBy, labelValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { LabelFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { LabelModelInfo, LabelModelLogic, TeamModelLogic } from "./types";

const __typename = "Label" as const;
export const LabelModel: LabelModelLogic = ({
    __typename,
    dbTable: "label",
    dbTranslationTable: "label_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, label: true }),
            get: (select) => select.label,
        },
    }),
    format: LabelFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                label: data.label,
                color: noNull(data.color),
                ownedByTeam: data.teamConnect ? { connect: { id: data.teamConnect } } : undefined,
                ownedByUser: !data.teamConnect ? { connect: { id: rest.userData.id } } : undefined,
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                label: noNull(data.label),
                color: noNull(data.color),
                apis: await shapeHelper({ relation: "apis", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Api", parentRelationshipName: "label", data, ...rest }),
                codes: await shapeHelper({ relation: "codes", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Code", parentRelationshipName: "label", data, ...rest }),
                focusModes: await shapeHelper({ relation: "focusModes", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "FocusMode", parentRelationshipName: "label", data, ...rest }),
                issues: await shapeHelper({ relation: "issues", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Issue", parentRelationshipName: "label", data, ...rest }),
                meetings: await shapeHelper({ relation: "meetings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Meeting", parentRelationshipName: "label", data, ...rest }),
                notes: await shapeHelper({ relation: "notes", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Note", parentRelationshipName: "label", data, ...rest }),
                projects: await shapeHelper({ relation: "projects", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Project", parentRelationshipName: "label", data, ...rest }),
                routines: await shapeHelper({ relation: "routines", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Routine", parentRelationshipName: "label", data, ...rest }),
                schedules: await shapeHelper({ relation: "schedules", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Schedule", parentRelationshipName: "label", data, ...rest }),
                standards: await shapeHelper({ relation: "standards", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Standard", parentRelationshipName: "label", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: labelValidation,
    },
    search: {
        defaultSort: LabelSortBy.DateUpdatedDesc,
        sortBy: LabelSortBy,
        searchFields: {
            createdTimeFrame: true,
            ownedByTeamId: true,
            ownedByUserId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["labelWrapped", "transDescriptionWrapped"] }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            ownedByTeam: "Team",
            ownedByUser: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Team: data?.ownedByTeam,
            User: data?.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (data, ...rest) =>
            (data.ownedByUser === null && data.ownedByTeam === null) ||
            oneIsPublic<LabelModelInfo["PrismaSelect"]>([
                ["ownedByTeam", "Team"],
                ["ownedByUser", "User"],
            ], data, ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { ownedByUser: { id: userId } },
                ],
            }),
        },
    }),
});
