import { PrismaType } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, OrganizationPermission, SessionUser } from '@shared/consts';
import { organizationValidation } from "@shared/validation";
import { Prisma, role } from "@prisma/client";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { ModelLogic } from "./types";
import { uuid } from "@shared/uuid";
import { getSingleTypePermissions } from "../validators";
import { noNull, onlyValidIds, shapeHelper } from "../builders";
import { bestLabel, tagShapeHelper, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { getLabels } from "../getters";

const __typename = 'Organization' as const;

const suppFields = ['isStarred', 'isViewed', 'permissionsOrganization', 'translatedName'] as const;
export const OrganizationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: OrganizationCreateInput,
    GqlUpdate: OrganizationUpdateInput,
    GqlModel: Organization,
    GqlSearch: OrganizationSearchInput,
    GqlSort: OrganizationSortBy,
    GqlPermission: OrganizationPermission,
    PrismaCreate: Prisma.organizationUpsertArgs['create'],
    PrismaUpdate: Prisma.organizationUpsertArgs['update'],
    PrismaModel: Prisma.organizationGetPayload<SelectWrap<Prisma.organizationSelect>>,
    PrismaSelect: Prisma.organizationSelect,
    PrismaWhere: Prisma.organizationWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.organization,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            apis: 'Api',
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'Organization',
            issues: 'Issue',
            labels: 'Label',
            meetings: 'Meeting',
            members: 'Member',
            notes: 'Note',
            parent: 'Organization',
            paymentHistory: 'Payment',
            posts: 'Post',
            premium: 'Premium',
            projects: 'Project',
            questions: 'Question',
            reports: 'Report',
            resourceList: 'ResourceList',
            roles: 'Role',
            routines: 'Routine',
            smartContracts: 'SmartContract',
            standards: 'Standard',
            starredBy: 'User',
            tags: 'Tag',
            transfersIncoming: 'Transfer',
            transfersOutgoing: 'Transfer',
            wallets: 'Wallet',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            directoryListings: 'ProjectVersionDirectory',
            issues: 'Issue',
            labels: 'Label',
            notes: 'Note',
            apis: 'Api',
            comments: 'Comment',
            forks: 'Organization',
            meetings: 'Meeting',
            parent: 'Organization',
            posts: 'Post',
            smartContracts: 'SmartContract',
            tags: 'Tag',
            members: 'Member',
            memberInvites: 'MemberInvite',
            projects: 'Project',
            questions: 'Question',
            reports: 'Report',
            resourceList: 'ResourceList',
            roles: 'Role',
            routines: 'Routine',
            runRoutines: 'RunRoutine',
            standards: 'Standard',
            starredBy: 'User',
            transfersIncoming: 'Transfer',
            transfersOutgoing: 'Transfer',
            wallets: 'Wallet',
        },
        joinMap: { starredBy: 'user', tags: 'tag' },
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
            toGraphQL: ({ ids, prisma, userData }) => ({
                isStarred: async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                isViewed: async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                permissionsOrganization: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
                translatedName: async () => await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'organization.translatedName'),
            }),
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => {
                // ID for organization and admin role
                const organizationId = uuid();
                const roleId = uuid();
                // Add "Admin" role to roles, and assign yourself to it
                const roles = await shapeHelper({ relation: 'roles', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'Role', parentRelationshipName: 'organization', data, prisma, userData })
                const adminRole = {
                    id: roleId,
                    name: 'Admin',
                    permissions: JSON.stringify({}), //TODO
                    members: {
                        create: {
                            permissions: JSON.stringify({}), //TODO
                            user: { connect: { id: userData.id } },
                            organization: { connect: { id: organizationId } },
                        }
                    }
                }
                if (roles.roles.create) roles.roles.create.push(adminRole);
                else roles.roles.create = [adminRole];
                // Add yourself as a member
                const adminMember = {
                    isAdmin: true,
                    permissions: JSON.stringify({}), //TODO
                    user: { connect: { id: userData.id } },
                    organization: { connect: { id: organizationId } },
                    roles: { connect: { id: roleId } },
                }
                return {
                    id: organizationId,
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
                    permissions: JSON.stringify({}), //TODO
                    createdBy: { connect: { id: userData.id } },
                    members: { create: adminMember },
                    ...(await shapeHelper({ relation: 'memberInvites', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'Member', parentRelationshipName: 'organization', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'organization', data, prisma, userData })),
                    ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
                    ...(await tagShapeHelper({ relTypes: ['Create'], parentType: 'Organization', relation: 'tags', data, prisma, userData })),
                    ...roles,
                }
            },
            update: async ({ data, prisma, userData }) => ({
                handle: noNull(data.handle),
                isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                isPrivate: noNull(data.isPrivate),
                ...(await shapeHelper({ relation: 'members', relTypes: ['Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Member', parentRelationshipName: 'organization', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'memberInvites', relTypes: ['Create', 'Update'], isOneToOne: false, isRequired: false, objectType: 'Member', parentRelationshipName: 'organization', data, prisma, userData })),
            })
        },
        yup: organizationValidation,
    },
    search: {
        defaultSort: OrganizationSortBy.StarsDesc,
        searchFields: {
            createdTimeFrame: true,
            isOpenToNewMembers: true,
            maxStars: true,
            maxViews: true,
            memberUserIds: true,
            minStars: true,
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
                'labelsWrapped',
                'tagsWrapped',
                'transNameWrapped',
                'transBioWrapped'
            ]
        })
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
        async hasRole(prisma: PrismaType, userId: string, organizationIds: (string | null | undefined)[], name: string = 'Admin'): Promise<Array<role | undefined>> {
            if (organizationIds.length === 0) return [];
            // Take out nulls
            const idsToQuery = onlyValidIds(organizationIds);
            // Query roles data for each organization ID
            const roles = await prisma.role.findMany({
                where: {
                    name,
                    organization: { id: { in: idsToQuery } },
                    members: { some: { user: { id: userId } } }
                }
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
        async findAdminInfo(prisma: PrismaType, organizationId: string, excludeUserId?: string): Promise<{ id: string, languages: string[] }[]> {
            const admins = await prisma.member.findMany({
                where: {
                    AND: [
                        { organizationId },
                        { isAdmin: true },
                        { userId: { not: excludeUserId } }
                    ]
                },
                select: {
                    user: {
                        select: {
                            id: true,
                            languages: { select: { language: true } }
                        },
                    }
                }
            })
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
        isTransferable: false,
        maxObjects: {
            User: {
                private: {
                    noPremium: 1,
                    premium: 10,
                },
                public: {
                    noPremium: 3,
                    premium: 25,
                }
            },
            Organization: 0,
        },
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
                                userId
                            },
                        }
                    },
                    select: {
                        permissions: true,
                    },
                },
                members: {
                    where: {
                        userId,
                    },
                    select: {
                        isAdmin: true,
                        permissions: true,
                    },
                }
            } : {}),
        }),
        permissionResolvers: ({ isAdmin, isPublic }) => ({
            // canAddMembers: async () => isAdmin,
            // canDelete: async () => isAdmin,
            // canEdit: async () => isAdmin,
            // canReport: async () => !isAdmin && isPublic,
            // canStar: async () => isAdmin || isPublic,
            // canView: async () => isAdmin || isPublic,
        } as any),
        owner: (data) => ({
            Organization: data,
        }),
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => OrganizationModel.query.hasRoleQuery(userId),
        }
        // if (!createMany && !updateMany && !deleteMany) return;
        // if (createMany) {
        //     createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
        // }
        // if (updateMany) {
        //     for (const input of updateMany) {
        //         await WalletModel.verify(prisma).verifyHandle('Organization', input.where.id, input.data.handle);
        //         lineBreaksCheck(input.data, ['bio'], 'LineBreaksBio');
        //     }
        // }
    },
})