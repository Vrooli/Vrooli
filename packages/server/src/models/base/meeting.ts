import { MaxObjects, MeetingSortBy, meetingValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { PreShapeEmbeddableTranslatableResult, labelShapeHelper, preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { MeetingFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MeetingModelLogic, TeamModelLogic } from "./types";

type MeetingPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Meeting" as const;
export const MeetingModel: MeetingModelLogic = ({
    __typename,
    dbTable: "meeting",
    dbTranslationTable: "meeting_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans?.description,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    }),
    format: MeetingFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<MeetingPre> => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as MeetingPre;
                return {
                    id: data.id,
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    showOnTeamProfile: noNull(data.showOnTeamProfile),
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "meetings", data, ...rest }),
                    restrictedToRoles: await shapeHelper({
                        relation: "restrictedToRoles", relTypes: ["Connect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                            fieldName: "role",
                            uniqueFieldName: "meeting_roles_meetingid_roleid_unique",
                            childIdFieldName: "roleId",
                            parentIdFieldName: "meetingId",
                            parentId: data.id ?? null,
                        }, data, ...rest,
                    }),
                    invites: await shapeHelper({ relation: "invites", relTypes: ["Create"], isOneToOne: false, objectType: "MeetingInvite", parentRelationshipName: "meeting", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "meetings", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Meeting", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as MeetingPre;
                return {
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    showOnTeamProfile: noNull(data.showOnTeamProfile),
                    restrictedToRoles: await shapeHelper({
                        relation: "restrictedToRoles", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                            fieldName: "role",
                            uniqueFieldName: "meeting_roles_meetingid_roleid_unique",
                            childIdFieldName: "roleId",
                            parentIdFieldName: "meetingId",
                            parentId: data.id ?? null,
                        }, data, ...rest,
                    }),
                    invites: await shapeHelper({ relation: "invites", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "MeetingInvite", parentRelationshipName: "meeting", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Connect", "Update", "Delete"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "meetings", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Create", "Update"], parentType: "Meeting", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsPlain({
                    ...params,
                    objectType: __typename,
                    ownerTeamField: "team",
                });
            },
        },
        yup: meetingValidation,
    },
    search: {
        defaultSort: MeetingSortBy.AttendeesDesc,
        sortBy: MeetingSortBy,
        searchFields: {
            createdTimeFrame: true,
            labelsIds: true,
            openToAnyoneWithInvite: true,
            scheduleEndTimeFrame: true,
            scheduleStartTimeFrame: true,
            showOnTeamProfile: true,
            teamId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "labelsWrapped",
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            dbFields: ["teamId"],
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
            showOnTeamProfile: true,
            team: "Team",
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canInvite: () => isLoggedIn && isAdmin,
        }),
        owner: (data) => ({
            Team: data?.team,
        }),
        isDeleted: () => false,
        isPublic: (data) => data.showOnTeamProfile === true,
        visibility: {
            private: function getVisibilityPrivate() {
                return {
                    OR: [
                        { showOnTeamProfile: false },
                        { team: { isPrivate: true } },
                    ],
                };
            },
            public: function getVisibilityPublic() {
                return {
                    showOnTeamProfile: true,
                    team: { isPrivate: false },
                };
            },
            owner: (userId) => ({
                team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId),
            }),
            attendingOrInvited: (userId) => ({
                OR: [
                    { attendees: { some: { user: { id: userId } } } },
                    { invites: { some: { userId } } },
                ],
            }),
        },
    }),
});
