import { PrismaType } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, ResourceListUsedFor, OrganizationPermission } from "../schema/types";
import { addJoinTablesHelper, removeJoinTablesHelper, addCountFieldsHelper, removeCountFieldsHelper, onlyValidIds, visibilityBuilder, combineQueries } from "./builder";
import { organizationsCreate, organizationsUpdate, organizationTranslationCreate, organizationTranslationUpdate } from "@shared/validation";
import { Prisma, role } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { ViewModel } from "./view";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { cudHelper } from "./actions";
import { Trigger } from "../events";
import { getSingleTypePermissions, MemberPolicy, OrganizationPolicy, RolePolicy } from "./validators";

const joinMapper = { starredBy: 'user', tags: 'tag' };
const countMapper = { commentsCount: 'comments', membersCount: 'members', reportsCount: 'reports' };
type SupplementalFields = 'isStarred' | 'isViewed' | 'permissionsOrganization';
export const organizationFormatter = (): FormatConverter<Organization, SupplementalFields> => ({
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
            ['isStarred', async () => await StarModel.query(prisma).getIsStarreds(userData?.id, ids, 'Organization')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userData?.id, ids, 'Organization')],
            ['permissionsOrganization', async () => await getSingleTypePermissions('Organization', ids, prisma, userData)],
        ],
    },
})

export const organizationSearcher = (): Searcher<
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

export const organizationValidator = (): Validator<
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
    isAdmin: (data, userId) => {
        // If no userId, can't be admin
        if (!userId) return false;
        // Convert stringified permissions to object
        const orgPermissions: OrganizationPolicy = asdfasfd;
        const rolePermissions: RolePolicy[] = fdsafdas;
        const memberPermissions: MemberPolicy | null = fdasfsd;
        return true;// TODO
    },
    isDeleted: () => false,
    isPublic: (data) => data.isPrivate === false,
    ownerOrMemberWhere: (userId) => organizationQuerier().hasRoleInOrganizationQuery(userId).organization,
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

export const organizationQuerier = () => ({
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

export const organizationMutater = (prisma: PrismaType): Mutater<Organization> => ({
    async shapeBase(userId: string, data: OrganizationCreateInput | OrganizationUpdateInput) {
        return {
            id: data.id,
            handle: data.handle ?? undefined,
            isOpenToNewMembers: data.isOpenToNewMembers ?? undefined,
            isPrivate: data.isPrivate ?? undefined,
            permissions: JSON.stringify({}),
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder!(userId, data, false),
            tags: await TagModel.mutate(prisma).tagRelationshipBuilder(userId, data, 'Organization'),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: organizationTranslationCreate, update: organizationTranslationUpdate }, false),
        }
    },
    async shapeCreate(userId: string, data: OrganizationCreateInput): Promise<Prisma.organizationUpsertArgs['create']> {
        // Add yourself as a member
        const members = { members: { create: { user: { connect: { id: userId } } } } };
        // Create an Admin role assignment for yourself
        const roles = { roles: { create: { title: 'Admin', assignees: { create: { user: { connect: { id: userId } } } } } } };
        return {
            ...members,
            ...roles,
            ...this.shapeBase(userId, data),
        }
    },
    async shapeUpdate(userId: string, data: OrganizationUpdateInput): Promise<Prisma.organizationUpsertArgs['update']> {
        // TODO members
        return {
            ...this.shapeBase(userId, data),
        }
    },
    async cud(params: CUDInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<CUDResult<Organization>> {
        return cudHelper({
            ...params,
            objectType: 'Organization',
            prisma,
            yup: { yupCreate: organizationsCreate, yupUpdate: organizationsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Organization', c.id as string, params.userData.id);
                }
            },
        })
    },
})

export const OrganizationModel = ({
    prismaObject: (prisma: PrismaType) => prisma.organization,
    format: organizationFormatter(),
    mutate: organizationMutater,
    search: organizationSearcher(),
    query: organizationQuerier,
    type: 'Organization' as GraphQLModelType,
    validate: organizationValidator(),
})