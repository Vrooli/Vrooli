import { MaxObjects, MeetingSortBy, getTranslation, meetingValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { afterMutationsPlain } from "../../utils/triggers/afterMutationsPlain.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { MeetingFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { MeetingModelInfo, MeetingModelLogic, TeamModelLogic } from "./types.js";

type MeetingPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Meeting" as const;
export const MeetingModel: MeetingModelLogic = ({
    __typename,
    dbTable: "meeting",
    dbTranslationTable: "meeting_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages?.[0]);
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
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            dbFields: ["teamId"],
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<MeetingModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
            own: function getOwn(data) {
                return {
                    team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Meeting", "Own", data),
                        useVisibility("Meeting", "Public", data),
                    ],
                };
            },
            // Search method not useful for this object because meetings are not explicitly set as private, so we'll return "Own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Meeting", "Own", data);
            },
            ownPublic: function getOwnPublic(data) {
                return useVisibility("Meeting", "Own", data);
            },
            public: function getVisibilityPublic() {
                return {
                    showOnTeamProfile: true,
                    team: { isPrivate: false },
                };
            },
            attendingOrInvited: function getAttendingOrInvited(data) {
                return {
                    OR: [
                        { attendees: { some: { user: { id: data.userId } } } },
                        { invites: { some: { user: { id: data.userId } } } },
                    ],
                };
            },
        },
    }),
});
