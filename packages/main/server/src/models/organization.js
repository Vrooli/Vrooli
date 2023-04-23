import { MaxObjects, OrganizationSortBy } from "@local/consts";
import { exists } from "@local/utils";
import { uuid } from "@local/uuid";
import { organizationValidation } from "@local/validation";
import { noNull, onlyValidIds, shapeHelper } from "../builders";
import { getLabels } from "../getters";
import { bestLabel, defaultPermissions, tagShapeHelper, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, handlesCheck, lineBreaksCheck } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ViewModel } from "./view";
const __typename = "Organization";
const suppFields = ["you", "translatedName"];
export const OrganizationModel = ({
    __typename,
    delegate: (prisma) => prisma.organization,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            apis: "Api",
            comments: "Comment",
            directoryListings: "ProjectVersionDirectory",
            forks: "Organization",
            issues: "Issue",
            labels: "Label",
            meetings: "Meeting",
            members: "Member",
            notes: "Note",
            parent: "Organization",
            paymentHistory: "Payment",
            posts: "Post",
            premium: "Premium",
            projects: "Project",
            questions: "Question",
            reports: "Report",
            resourceList: "ResourceList",
            roles: "Role",
            routines: "Routine",
            smartContracts: "SmartContract",
            standards: "Standard",
            bookmarkedBy: "User",
            tags: "Tag",
            transfersIncoming: "Transfer",
            transfersOutgoing: "Transfer",
            wallets: "Wallet",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            directoryListings: "ProjectVersionDirectory",
            issues: "Issue",
            labels: "Label",
            notes: "Note",
            apis: "Api",
            comments: "Comment",
            forks: "Organization",
            meetings: "Meeting",
            parent: "Organization",
            posts: "Post",
            smartContracts: "SmartContract",
            tags: "Tag",
            members: "Member",
            memberInvites: "MemberInvite",
            projects: "Project",
            questions: "Question",
            reports: "Report",
            resourceList: "ResourceList",
            roles: "Role",
            routines: "Routine",
            runRoutines: "RunRoutine",
            standards: "Standard",
            bookmarkedBy: "User",
            transfersIncoming: "Transfer",
            transfersOutgoing: "Transfer",
            wallets: "Wallet",
        },
        joinMap: { bookmarkedBy: "user", tags: "tag" },
        countFields: {
            apisCount: true,
            commentsCount: true,
            issuesCount: true,
            labelsCount: true,
            meetingsCount: true,
            membersCount: true,
            notesCount: true,
            postsCount: true,
            projectsCount: true,
            questionsCount: true,
            reportsCount: true,
            rolesCount: true,
            routinesCount: true,
            smartContractsCount: true,
            standardsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, prisma, userData?.languages ?? ["en"], "organization.translatedName"),
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList, prisma, userData }) => {
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ["bio"], "LineBreaksBio", userData.languages));
                const handleData = updateList.map(({ data, where }) => ({ id: where.id, handle: data.handle }));
                await handlesCheck(prisma, "Organization", handleData, userData.languages);
            },
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
                    permissions: JSON.stringify({}),
                    createdBy: { connect: { id: rest.userData.id } },
                    members: {
                        create: {
                            isAdmin: true,
                            permissions: JSON.stringify({}),
                            user: { connect: { id: rest.userData.id } },
                        },
                    },
                    ...(await shapeHelper({ relation: "memberInvites", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "organization", data, ...rest })),
                    ...(await shapeHelper({ relation: "roles", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "organization", data, ...rest })),
                    ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
                    ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Organization", relation: "tags", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => ({
                handle: noNull(data.handle),
                isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                isPrivate: noNull(data.isPrivate),
                permissions: JSON.stringify({}),
                ...(await shapeHelper({ relation: "members", relTypes: ["Delete"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "memberInvites", relTypes: ["Create", "Delete"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "organization", data, ...rest })),
                ...(await shapeHelper({ relation: "roles", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "organization", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Organization", relation: "tags", data, ...rest })),
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
                for (const { id: organizationId } of created) {
                    await prisma.role.create({
                        data: {
                            id: uuid(),
                            name: "Admin",
                            permissions: JSON.stringify({}),
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
            visibility: true,
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
    },
    query: {
        hasRoleQuery: (userId, names = []) => {
            if (names.length === 0)
                return { members: { some: { user: { id: userId } } } };
            return { roles: { some: { name: { in: names }, members: { some: { user: { id: userId } } } } } };
        },
        isMemberOfOrganizationQuery: (userId) => ({ ownedByOrganization: { members: { some: { userId } } } }),
        async hasRole(prisma, userId, organizationIds, name = "Admin") {
            if (organizationIds.length === 0)
                return [];
            if (!exists(userId))
                return organizationIds.map(() => undefined);
            const idsToQuery = onlyValidIds(organizationIds);
            const roles = await prisma.role.findMany({
                where: {
                    name,
                    organization: { id: { in: idsToQuery } },
                    members: { some: { user: { id: userId } } },
                },
            });
            return organizationIds.map(id => roles.find(({ organizationId }) => organizationId === id));
        },
        async findAdminInfo(prisma, organizationId, excludeUserId) {
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
            const result = [];
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
//# sourceMappingURL=organization.js.map