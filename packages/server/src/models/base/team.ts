import { MaxObjects, TeamSortBy, exists, teamValidation, uuid } from "@local/shared";
import { role } from "@prisma/client";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { shapeHelper } from "../../builders/shapeHelper";
import { prismaInstance } from "../../db/instance";
import { getLabels } from "../../getters";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { PreShapeEmbeddableTranslatableResult, preShapeEmbeddableTranslatable, tagShapeHelper, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, handlesCheck, lineBreaksCheck } from "../../validators";
import { TeamFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, TeamModelLogic, ViewModelLogic } from "./types";

type TeamPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Team" as const;
export const TeamModel: TeamModelLogic = ({
    __typename,
    dbTable: "team",
    dbTranslationTable: "team_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, bio: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    bio: trans?.bio,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    }),
    format: TeamFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, userData }): Promise<TeamPre> => {
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["bio"], "LineBreaksBio", userData.languages));
                await handlesCheck(__typename, Create, Update, userData.languages);
                // Find translations that need text embeddings
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    id: data.id,
                    bannerImage: data.bannerImage,
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
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "team", data, ...rest }),
                    roles: await shapeHelper({ relation: "roles", relTypes: ["Create"], isOneToOne: false, objectType: "Role", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    bannerImage: noNull(data.bannerImage),
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
                    permissions: JSON.stringify({}), //TODO
                    profileImage: noNull(data.profileImage),
                    members: await shapeHelper({ relation: "members", relTypes: ["Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    memberInvites: await shapeHelper({ relation: "memberInvites", relTypes: ["Create", "Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "team", data, ...rest }),
                    roles: await shapeHelper({ relation: "roles", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "Role", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                for (const teamId of createdIds) {
                    // Upsert "Admin" role (in case they already included it in the request). 
                    // Trying to connect you as a member again shouldn't throw an error (hopefully)
                    await prismaInstance.role.upsert({
                        where: {
                            role_teamId_name_unique: {
                                name: "Admin",
                                teamId,
                            },
                        },
                        create: {
                            id: uuid(),
                            name: "Admin",
                            permissions: JSON.stringify({}), //TODO
                            members: {
                                connect: {
                                    member_teamid_userid_unique: {
                                        userId: userData.id,
                                        teamId,
                                    },
                                },
                            },
                            teamId,
                        },
                        update: {
                            permissions: JSON.stringify({}), //TODO
                            members: {
                                connect: {
                                    member_teamid_userid_unique: {
                                        userId: userData.id,
                                        teamId,
                                    },
                                },
                            },
                        },
                    });
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
                "labelsOwnerWrapped",
                "tagsWrapped",
                "transNameWrapped",
                "transBioWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, userData?.languages ?? ["en"], "team.translatedName"),
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
            const roles = await prismaInstance.role.findMany({
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
            const admins = await prismaInstance.member.findMany({
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
                roles: {
                    where: {
                        members: {
                            some: {
                                userId,
                            },
                        },
                    },
                    select: {
                        id: true,
                        permissions: true,
                    },
                },
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
            private: function getVisibilityPrivate() {
                return {
                    isPrivate: true,
                };
            },
            public: function getVisibilityPublic() {
                return {
                    isPrivate: false,
                };
            },
            owner: (userId) => ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId),
        },
    }),
});
