import { DEFAULT_LANGUAGE, MaxObjects, TeamSortBy, exists, getTranslation, teamValidation } from "@local/shared";
import { role } from "@prisma/client";
import { noNull } from "../../builders/noNull.js";
import { onlyValidIds } from "../../builders/onlyValid.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { getLabels } from "../../getters/getLabels.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { tagShapeHelper } from "../../utils/shapes/tagShapeHelper.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { handlesCheck } from "../../validators/handlesCheck.js";
import { lineBreaksCheck } from "../../validators/lineBreaksCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { TeamFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, TeamModelInfo, TeamModelLogic, ViewModelLogic } from "./types.js";

type TeamPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Team" as const;
export const TeamModel: TeamModelLogic = ({
    __typename,
    dbTable: "team",
    dbTranslationTable: "team_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, bio: true } } }),
            get: ({ translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    bio: trans.bio,
                    name: trans.name,
                }, languages?.[0]);
            },
        },
    }),
    format: TeamFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<TeamPre> => {
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["bio"], "LineBreaksBio"));
                await handlesCheck(__typename, Create, Update);
                // Find translations that need text embeddings
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    id: data.id,
                    bannerImage: data.bannerImage,
                    config: noNull(data.config),
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: data.isPrivate,
                    permissions: JSON.stringify({}), //TODO
                    profileImage: data.profileImage,
                    createdBy: { connect: { id: rest.userData.id } },
                    members: {
                        create: {
                            isAdmin: true,
                            permissions: JSON.stringify({}), //TODO
                            user: { connect: { id: rest.userData.id } },
                        },
                    },
                    memberInvites: await shapeHelper({ relation: "memberInvites", relTypes: ["Create"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    bannerImage: noNull(data.bannerImage),
                    config: noNull(data.config),
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
                    permissions: JSON.stringify({}), //TODO
                    profileImage: noNull(data.profileImage),
                    members: await shapeHelper({ relation: "members", relTypes: ["Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    memberInvites: await shapeHelper({ relation: "memberInvites", relTypes: ["Create", "Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                for (const teamId of createdIds) {
                    // Handle trigger
                    // Trigger(userData.languages).createTeam(userData.id, teamId);
                }
            },
        },
        yup: teamValidation,
    },
    search: {
        defaultSort: TeamSortBy.BookmarksDesc,
        searchFields: {
            createdTimeFrame: true,
            isOpenToNewMembers: true,
            maxBookmarks: true,
            maxViews: true,
            memberUserIds: true,
            minBookmarks: true,
            minViews: true,
            projectId: true,
            reportId: true,
            routineId: true,
            standardId: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: TeamSortBy,
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                "transNameWrapped",
                "transBioWrapped",
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<TeamModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, userData?.languages ?? [DEFAULT_LANGUAGE], "team.translatedName"),
                };
            },
        },
    },
    query: {
        /**
         * Query for checking if a user has a specific role in a team
         */
        hasRoleQuery: (userId: string, names: string[] = []) => {
            // If no names are provided, query by members (in case user was not assigned a role)
            if (names.length === 0) return { members: { some: { user: { id: userId } } } };
            // Otherwise, query for roles with one of the provided names
            return { roles: { some: { name: { in: names }, members: { some: { user: { id: userId } } } } } };
        },
        /**
         * Query for checking if a user is a member of a team
         */
        isMemberOfTeamQuery: (userId: string) => (
            { ownedByTeam: { members: { some: { userId } } } }
        ),
        /**
         * Determines if a user has a role of a list of teams
         * @param userId The user's ID
         * @param teamIds The list of team IDs
         * @param name The name of the role
         * @returns Array in the same order as the ids, with either admin/owner role data or undefined
         */
        async hasRole(userId: string | null | undefined, teamIds: (string | null | undefined)[], name = "Admin"): Promise<Array<role | undefined>> {
            if (teamIds.length === 0) return [];
            if (!exists(userId)) return teamIds.map(() => undefined);
            // Take out nulls
            const idsToQuery = onlyValidIds(teamIds);
            // Query roles data for each team ID
            const roles = await DbProvider.get().role.findMany({
                where: {
                    name,
                    team: { id: { in: idsToQuery } },
                    members: { some: { user: { id: userId } } },
                },
            });
            // Create an array of the same length as the input, with the role data or undefined
            return teamIds.map(id => roles.find(({ teamId }) => teamId === id));
        },
        /**
         * Finds every admin of a team
         * @param teamId The team ID
         * @param excludedUsers IDs of users to exclude from results
         * @returns A list of admin ids and their preferred languages. Useful for sending notifications
         */
        async findAdminInfo(teamId: string, excludedUsers?: string[] | string): Promise<{ id: string, languages: string[] }[]> {
            const admins = await DbProvider.get().member.findMany({
                where: {
                    AND: [
                        { teamId },
                        { isAdmin: true },
                        ...(typeof excludedUsers === "string" ?
                            [{ userId: { not: excludedUsers } }] :
                            Array.isArray(excludedUsers) ?
                                [{ userId: { notIn: excludedUsers } }] :
                                []
                        ),
                    ],
                },
                select: {
                    user: {
                        select: {
                            id: true,
                            languages: { select: { language: true } },
                        },
                    },
                },
            });
            const result: { id: string, languages: string[] }[] = [];
            admins.forEach(({ user }) => {
                result.push({
                    id: user.id,
                    languages: user.languages.map(({ language }) => language),
                });
            });
            return result;
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canAddMembers: () => isLoggedIn && isAdmin,
        }),
        permissionsSelect: (userId) => ({
            id: true,
            isOpenToNewMembers: true,
            isPrivate: true,
            languages: { select: { language: true } },
            permissions: true,
            ...(userId ? {
                members: {
                    where: {
                        userId,
                    },
                    select: {
                        id: true,
                        isAdmin: true,
                        permissions: true,
                        userId: true,
                    },
                },
            } : {}),
        }),
        visibility: {
            own: function getOwn(data) {
                return ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId);
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        // Owned objects
                        ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId),
                        // Public objects
                        useVisibility("Team", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    ...ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId),
                    isPrivate: true,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    ...ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId),
                    isPrivate: false,
                };
            },
            public: function getPublic() {
                return {
                    isPrivate: false,
                };
            },
        },
    }),
});
