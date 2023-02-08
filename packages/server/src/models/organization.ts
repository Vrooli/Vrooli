import { PrismaType } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, OrganizationYou, PrependString } from '@shared/consts';
import { organizationValidation } from "@shared/validation";
import { Prisma, role } from "@prisma/client";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { ModelLogic } from "./types";
import { uuid } from "@shared/uuid";
import { getSingleTypePermissions, lineBreaksCheck, handlesCheck } from "../validators";
import { noNull, onlyValidIds, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, tagShapeHelper, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { getLabels } from "../getters";

const __typename = 'Organization' as const;
type Permissions = Pick<OrganizationYou, 'canAddMembers' | 'canDelete' | 'canUpdate' | 'canStar' | 'canRead'>;
const suppFields = ['you.canAddMembers', 'you.canDelete', 'you.canUpdate', 'you.canStar', 'you.canRead', 'you.isStarred', 'you.isViewed', 'translatedName'] as const;
export const OrganizationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: OrganizationCreateInput,
    GqlUpdate: OrganizationUpdateInput,
    GqlModel: Organization,
    GqlSearch: OrganizationSearchInput,
    GqlSort: OrganizationSortBy,
    GqlPermission: Permissions,
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
            toGraphQL: async ({ ids, prisma, userData }) => {
                console.log('organization tographql 1');
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                console.log('organization tographql 2', JSON.stringify(permissions), '\n');
                console.log('organization tographql 3', JSON.stringify((Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>)), '\n');
                let temp1 = await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename);
                console.log('organization tographql 4', JSON.stringify(temp1), '\n');
                let temp2 = await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename);
                console.log('organization tographql 5', JSON.stringify(temp2), '\n');
                let temp3 = await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'project.translatedName');
                console.log('organization tographql 6', JSON.stringify(temp3), '\n');
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                    'you.isViewed': await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    'translatedName': await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'project.translatedName')
                }
            },
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
                    ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Organization', relation: 'tags', data, prisma, userData })),
                    ...roles,
                }
            },
            update: async ({ data, prisma, userData }) => ({
                handle: noNull(data.handle),
                isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                isPrivate: noNull(data.isPrivate),
                ...(await shapeHelper({ relation: 'members', relTypes: ['Delete'], isOneToOne: false, isRequired: false, objectType: 'Member', parentRelationshipName: 'organization', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'memberInvites', relTypes: ['Create', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'Member', parentRelationshipName: 'organization', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'organization', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Organization', relation: 'tags', data, prisma, userData })),
                //...roles //TODO
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
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
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
        owner: (data) => ({
            Organization: data,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isPublic }),
            canAddMembers: () => isAdmin,
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
        validations: {
            create({ createMany, languages }) {
                createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio', languages))
            },
            async update({ languages, prisma, updateMany }) {
                // Validate AdaHandles
                let handleData = updateMany.map(({ data, where }) => ({ id: where.id, handle: data.handle })) as { id: string, handle: string | null | undefined }[];
                await handlesCheck(prisma, 'Organization', handleData, languages);
                // Validate line breaks
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['bio'], 'LineBreaksBio', languages));
            },
        },
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => OrganizationModel.query.hasRoleQuery(userId),
        },
    },
})