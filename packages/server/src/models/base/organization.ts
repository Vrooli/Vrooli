import { exists, MaxObjects, OrganizationSortBy, organizationValidation, uuid } from "@local/shared";
import { role } from "@prisma/client";
import { noNull, onlyValidIds, shapeHelper } from "../../builders";
import { getLabels } from "../../getters";
import { PrismaType } from "../../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, tagShapeHelper, translationShapeHelper } from "../../utils";
import { preShapeEmbeddableTranslatable } from "../../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions, handlesCheck, lineBreaksCheck } from "../../validators";
import { OrganizationFormat } from "../format/organization";
import { ModelLogic } from "../types";
import { BookmarkModel } from "./bookmark";
import { OrganizationModelLogic } from "./types";
import { ViewModel } from "./view";

const __typename = "Organization" as const;
const suppFields = ["you", "translatedName"] as const;
export const OrganizationModel: ModelLogic<OrganizationModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.organization,
    display: {
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
    },
    format: OrganizationFormat,
    mutate: {
        shape: {
            pre: async ({ createList, updateList, prisma, userData }) => {
                [...createList, ...updateList].forEach(input => lineBreaksCheck(input, ["bio"], "LineBreaksBio", userData.languages));
                // Validate AdaHandles
                const handleData = updateList.map(u => ({ id: u.id, handle: u.handle })) as { id: string, handle: string | null | undefined }[];
                await handlesCheck(prisma, "Organization", handleData, userData.languages);
                // Find translations that need text embeddings
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    bannerImage: data.bannerImage,
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
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
                    ...(await shapeHelper({ relation: "memberInvites", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "organization", data, ...rest })),
                    ...(await shapeHelper({ relation: "roles", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "organization", data, ...rest })),
                    ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Organization", relation: "tags", data, ...rest })),
                    ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => ({
                bannerImage: noNull(data.bannerImage),
                handle: noNull(data.handle),
                isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                isPrivate: noNull(data.isPrivate),
                permissions: JSON.stringify({}), //TODO
                profileImage: noNull(data.profileImage),
                ...(await shapeHelper({ relation: "members", relTypes: ["Delete"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "memberInvites", relTypes: ["Create", "Delete"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "roles", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "organization", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Organization", relation: "tags", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
                for (const { id: organizationId } of created) {
                    // Add 'Admin' role to organization
                    await prisma.role.create({
                        data: {
                            id: uuid(),
                            name: "Admin",
                            permissions: JSON.stringify({}), //TODO
                            members: {
                                connect: {
                                    member_organizationid_userid_unique: {
                                        userId: userData.id,
                                        organizationId,
                                    },
                                },
                            },
                            organizationId,
                        },
                    });
                    // Handle trigger
                    // Trigger(prisma, userData.languages).createOrganization(userData.id, organizationId);
                }
            },
        },
        yup: organizationValidation,
    },
    search: {
        defaultSort: OrganizationSortBy.BookmarksDesc,
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
        sortBy: OrganizationSortBy,
        searchStringQuery: () => ({
            OR: [
                "labelsOwnerWrapped",
                "tagsWrapped",
                "transNameWrapped",
                "transBioWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, prisma, userData?.languages ?? ["en"], "organization.translatedName"),
                };
            },
        },
    },
    query: {
        /**
         * Query for checking if a user has a specific role in an organization
         */
        hasRoleQuery: (userId: string, names: string[] = []) => {
            // If no names are provided, query by members (in case user was not assigned a role)
            if (names.length === 0) return { members: { some: { user: { id: userId } } } };
            // Otherwise, query for roles with one of the provided names
            return { roles: { some: { name: { in: names }, members: { some: { user: { id: userId } } } } } };
        },
        /**
         * Query for checking if a user is a member of an organization
         */
        isMemberOfOrganizationQuery: (userId: string) => (
            { ownedByOrganization: { members: { some: { userId } } } }
        ),
        /**
         * Determines if a user has a role of a list of organizations
         * @param userId The user's ID
         * @param organizationIds The list of organization IDs
         * @param name The name of the role
         * @returns Array in the same order as the ids, with either admin/owner role data or undefined
         */
        async hasRole(prisma: PrismaType, userId: string | null | undefined, organizationIds: (string | null | undefined)[], name = "Admin"): Promise<Array<role | undefined>> {
            if (organizationIds.length === 0) return [];
            if (!exists(userId)) return organizationIds.map(() => undefined);
            // Take out nulls
            const idsToQuery = onlyValidIds(organizationIds);
            // Query roles data for each organization ID
            const roles = await prisma.role.findMany({
                where: {
                    name,
                    organization: { id: { in: idsToQuery } },
                    members: { some: { user: { id: userId } } },
                },
            });
            // Create an array of the same length as the input, with the role data or undefined
            return organizationIds.map(id => roles.find(({ organizationId }) => organizationId === id));
        },
        /**
         * Finds every admin of an organization
         * @param prisma The prisma client
         * @param organizationId The organization ID
         * @param excludeUserId An option user to exclude from the results
         * @returns A list of admin ids and their preferred languages. Useful for sending notifications
         */
        async findAdminInfo(prisma: PrismaType, organizationId: string, excludeUserId?: string | null | undefined): Promise<{ id: string, languages: string[] }[]> {
            const admins = await prisma.member.findMany({
                where: {
                    AND: [
                        { organizationId },
                        { isAdmin: true },
                        { userId: { not: excludeUserId ?? undefined } },
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
    validate: {
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data,
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
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => OrganizationModel.query.hasRoleQuery(userId),
        },
    },
});
