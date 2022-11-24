import { PrismaType } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, ResourceListUsedFor, OrganizationPermission, SessionUser } from "../schema/types";
import { addJoinTablesHelper, removeJoinTablesHelper, addCountFieldsHelper, removeCountFieldsHelper, onlyValidIds, visibilityBuilder, combineQueries } from "./builder";
import { handle, organizationsCreate, organizationsUpdate } from "@shared/validation";
import { Prisma, role } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater, Displayer } from "./types";
import { getSingleTypePermissions } from "./validators";
import { uuid } from "@shared/uuid";
import { relBuilderHelper } from "./actions";
import { bestLabel, translationRelationshipBuilder } from "./utils";

const joinMapper = { starredBy: 'user', tags: 'tag' };
const countMapper = { commentsCount: 'comments', membersCount: 'members', reportsCount: 'reports' };
type SupplementalFields = 'isStarred' | 'isViewed' | 'permissionsOrganization';
const formatter = (): Formatter<Organization, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Organization',
        comments: 'Comment',
        members: 'Member',
        projects: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        routines: 'Routine',
        routinesCreated: 'Routine',
        starredBy: 'User',
        tags: 'Tag',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isViewed', 'permissionsOrganization'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Organization')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'Organization')],
            ['permissionsOrganization', async () => await getSingleTypePermissions('Organization', ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<
    OrganizationSearchInput,
    OrganizationSortBy,
    Prisma.organizationOrderByWithRelationInput,
    Prisma.organizationWhereInput
> => ({
    defaultSort: OrganizationSortBy.StarsDesc,
    sortMap: {
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { stars: 'asc' },
        StarsDesc: { stars: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
            { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
            { tags: { some: { tag: { tag: { ...insensitive } } } } },
        ]
    }),
    customQueries(input, userId) {
        return combineQueries([
            visibilityBuilder({ model: OrganizationModel, userId, visibility: input.visibility }),
            (input.isOpenToNewMembers !== undefined ? { isOpenToNewMembers: true } : {}),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.projectId !== undefined ? { projects: { some: { projectId: input.projectId } } } : {}),
            (input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            (input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            (input.userId !== undefined ? { members: { some: { userId: input.userId } } } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.standardId !== undefined ? { standards: { some: { id: input.standardId } } } : {}),
            (input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        ])
    },
})

const validator = (): Validator<
    OrganizationCreateInput,
    OrganizationUpdateInput,
    Organization,
    Prisma.organizationGetPayload<{ select: { [K in keyof Required<Prisma.organizationSelect>]: true } }>,
    OrganizationPermission,
    Prisma.organizationSelect,
    Prisma.organizationWhereInput
> => ({
    validateMap: {
        __typename: 'Organization',
        members: 'Member',
        projects: 'Project',
        routines: 'Routine',
        wallets: 'Wallet',
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
                    assignees: {
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
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canAddMembers', async () => isAdmin],
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canReport', async () => !isAdmin && isPublic],
        ['canStar', async () => isAdmin || isPublic],
        ['canView', async () => isAdmin || isPublic],
    ]),
    owner: (data) => ({
        Organization: data,
    }),
    isDeleted: () => false,
    isPublic: (data) => data.isPrivate === false,
    ownerOrMemberWhere: (userId) => querier().hasRoleInOrganizationQuery(userId).organization,
    // if (!createMany && !updateMany && !deleteMany) return;
    // if (createMany) {
    //     createMany.forEach(input => lineBreaksCheck(input, ['bio'], CODE.LineBreaksBio));
    // }
    // if (updateMany) {
    //     for (const input of updateMany) {
    //         await WalletModel.verify(prisma).verifyHandle('Organization', input.where.id, input.data.handle);
    //         lineBreaksCheck(input.data, ['bio'], CODE.LineBreaksBio);
    //     }
    // }
})

const querier = () => ({
    /**
     * Query for checking if a user has a specific role in an organization
     */
    hasRoleInOrganizationQuery: (userId: string, titles: string[] = []) => {
        // If no titles are provided, query by members (in case user was not assigned a role)
        if (titles.length === 0) return { organization: { members: { some: { user: { id: userId } } } } };
        // Otherwise, query for roles with one of the provided titles
        return { organization: { roles: { some: { title: { in: titles }, assignees: { some: { user: { id: userId } } } } } } }
    },
    /**
     * Query for checking if a user is a member of an organization
     */
    isMemberOfOrganizationQuery: (userId: string) => (
        { organization: { members: { some: { userId } } } }
    ),
    /**
     * Determines if a user has a role of a list of organizations
     * @param userId The user's ID
     * @param organizationIds The list of organization IDs
     * @param title The name of the role
     * @returns Array in the same order as the ids, with either admin/owner role data or undefined
     */
    async hasRole(prisma: PrismaType, userId: string, organizationIds: (string | null | undefined)[], title: string = 'Admin'): Promise<Array<role | undefined>> {
        if (organizationIds.length === 0) return [];
        // Take out nulls
        const idsToQuery = onlyValidIds(organizationIds);
        // Query roles data for each organization ID
        const roles = await prisma.role.findMany({
            where: {
                title,
                organization: { id: { in: idsToQuery } },
                assignees: { some: { user: { id: userId } } }
            }
        });
        // Create an array of the same length as the input, with the role data or undefined
        return organizationIds.map(id => roles.find(({ organizationId }) => organizationId === id));
    },
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: OrganizationCreateInput | OrganizationUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        handle: data.handle ?? undefined,
        isOpenToNewMembers: data.isOpenToNewMembers ?? undefined,
        isPrivate: data.isPrivate ?? undefined,
        permissions: JSON.stringify({}),
        resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        tags: await TagModel.mutate.relationshipBuilder!(prisma, userData.id, data, 'Organization'),
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

const mutater = (): Mutater<
    Organization,
    { graphql: OrganizationCreateInput, db: Prisma.organizationUpsertArgs['create'] },
    { graphql: OrganizationUpdateInput, db: Prisma.organizationUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            // ID for your member record
            const memberId = uuid();
            return {
                ...(await shapeBase(prisma, userData, data, true)),
                // Add yourself as a member
                members: {
                    create: {
                        id: memberId,
                        permissions: JSON.stringify({}), //TODO
                        user: { connect: { id: userData.id } }
                    }
                },
                // Give yourself an Admin role
                roles: {
                    create: {
                        title: 'Admin',
                        permissions: JSON.stringify({}), //TODO
                        assignees: { connect: { id: memberId } }
                    }
                }
            }
        },
        update: async ({ data, prisma, userData }) => {
            // TODO members
            return {
                ...(await shapeBase(prisma, userData, data, false)),
            }
        }
    },
    yup: { create: organizationsCreate, update: organizationsUpdate },
})

const displayer = (): Displayer => ({
    labels: async (prisma, objects) => {
        const translations = await prisma.organization_translation.findMany({
            where: { organizationId: { in: objects.map((o) => o.id) } },
            select: { organizationId: true, language: true, name: true }    
        })
        return objects.map(o => {
            const oTrans = translations.filter(t => t.organizationId === o.id);
            return bestLabel(oTrans, 'name', o.languages);
        })
    }
})

export const OrganizationModel = ({
    delegate: (prisma: PrismaType) => prisma.organization,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    query: querier(),
    type: 'Organization' as GraphQLModelType,
    validate: validator(),
})