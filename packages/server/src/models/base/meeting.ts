import { MaxObjects, MeetingSortBy, meetingValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { labelShapeHelper, preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { MeetingFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MeetingModelLogic, OrganizationModelLogic } from "./types";

const __typename = "Meeting" as const;
export const MeetingModel: MeetingModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.meeting,
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
            pre: async ({ Create, Update }) => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                showOnOrganizationProfile: noNull(data.showOnOrganizationProfile),
                organization: await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, objectType: "Organization", parentRelationshipName: "meetings", data, ...rest }),
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
                translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                showOnOrganizationProfile: noNull(data.showOnOrganizationProfile),
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
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: "organization",
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
            organizationId: true,
            scheduleEndTimeFrame: true,
            scheduleStartTimeFrame: true,
            showOnOrganizationProfile: true,
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
            dbFields: ["organizationId"],
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
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            showOnOrganizationProfile: true,
            organization: "Organization",
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canInvite: () => isLoggedIn && isAdmin,
        }),
        owner: (data) => ({
            Organization: data?.organization,
        }),
        isDeleted: () => false,
        isPublic: (data) => data.showOnOrganizationProfile === true,
        visibility: {
            private: {
                OR: [
                    { showOnOrganizationProfile: false },
                    { organization: { isPrivate: true } },
                ],
            },
            public: {
                AND: [
                    { showOnOrganizationProfile: true },
                    { organization: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                organization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId),
            }),
        },
    }),
});
