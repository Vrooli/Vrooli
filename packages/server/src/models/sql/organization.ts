import { PrismaType, RecursivePartial } from "../../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Count, ResourceListUsedFor, OrganizationPermission } from "../../schema/types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, removeJoinTablesHelper, Searcher, selectHelper, modelToGraphQL, ValidateMutationsInput, addCountFieldsHelper, removeCountFieldsHelper, addSupplementalFieldsHelper, Permissioner, getSearchStringQueryHelper, onlyValidIds, visibilityBuilder, combineQueries, onlyValidHandles } from "./base";
import { CustomError } from "../../error";
import { organizationsCreate, organizationsUpdate, organizationTranslationCreate, organizationTranslationUpdate } from "@shared/validation";
import { CODE } from '@shared/consts';
import { omit } from '@shared/utils';
import { role } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { WalletModel } from "./wallet";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";
import { GraphQLModelType } from ".";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user', tags: 'tag' };
const countMapper = { commentsCount: 'comments', membersCount: 'members', reportsCount: 'reports' };
const supplementalFields = ['isStarred', 'isViewed', 'permissionsOrganization'];
export const organizationFormatter = (): FormatConverter<Organization, OrganizationPermission> => ({
    relationshipMap: {
        '__typename': 'Organization',
        'comments': 'Comment',
        'members': 'Member',
        'projects': 'Project',
        'reports': 'Report',
        'resourceLists': 'ResourceList',
        'routines': 'Routine',
        'routinesCreated': 'Routine',
        'starredBy': 'User',
        'tags': 'Tag',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    removeSupplementalFields: (partial) => omit(partial, supplementalFields),
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Organization>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Organization')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Organization')],
                ['permissionsOrganization', async () => await OrganizationModel.permissions().get({ objects, permissions, prisma, userId })],
            ]
        });
    }
})

export const organizationPermissioner = (): Permissioner<OrganizationPermission, OrganizationSearchInput> => ({
    async get({
        objects,
        permissions,
        prisma,
        userId,
    }) {
        // Initialize result with default permissions
        const result: (OrganizationPermission & { id?: string, handle?: string })[] = objects.map((o) => ({
            handle: o.handle,
            id: o.id,
            canAddMembers: false, // own, even if isOpenToNewMembers is false or isPrivate is true
            canDelete: false, // own
            canEdit: false, // own
            canReport: true, // !own && !isPrivate
            canStar: false, // own || !isPrivate
            canView: false, // own || !isPrivate
            isMember: false, // is member, not necessarily owner
        }));
        // Check ownership
        if (userId) {
            // Query for all roles
            const roles = await prisma.role.findMany({
                where: {
                    organization: {
                        OR: [
                            { id: { in: onlyValidIds(objects.map((o) => o.id)) } },
                            { handle: { in: onlyValidHandles(objects.map((o) => o.handle)) } },
                        ]
                    },
                    assignees: { some: { user: { id: userId } } }
                },
                select: { title: true, organizationId: true }
            });
            // Loop through roles to set permissions.
            // A role with title 'Admin' means you are the owner. Other roles mean 
            // you are a member.
            for (const role of roles) {
                const index = result.findIndex((r) => r.id === role.organizationId);
                if (index >= 0) {
                    result[index] = {
                        ...result[index],
                        canAddMembers: result[index].canAddMembers || role.title === 'Admin',
                        canDelete: result[index].canDelete || role.title === 'Admin',
                        canEdit: result[index].canEdit || role.title === 'Admin',
                        canReport: result[index].canReport === false ? false : role.title !== 'Admin',
                        canStar: result[index].canStar || role.title === 'Admin',
                        canView: result[index].canView || role.title === 'Admin',
                        isMember: true,
                    }
                }
            }
        }
        // Query all objects
        const all = await prisma.organization.findMany({
            where: {
                OR: [
                    { id: { in: onlyValidIds(objects.map((o) => o.id)) } },
                    { handle: { in: onlyValidHandles(objects.map((o) => o.handle)) } },
                ]
            },
            select: { id: true, isPrivate: true },
        })
        // Set permissions for all objects
        all.forEach((o) => {
            const index = objects.findIndex((r) => r.id === o.id);
            result[index] = {
                ...result[index],
                canReport: result[index].canReport === false ? false : !o.isPrivate,
                canStar: result[index].canStar || !o.isPrivate,
                canView: result[index].canView || !o.isPrivate,
            }
        });
        // Return result with IDs and handles removed
        result.forEach((r) => delete r.id && delete r.handle);
        return result as OrganizationPermission[];
    },
    async canSearch({
        input,
        prisma,
        userId
    }) {
        //TODO
        return 'full';
    },
    ownershipQuery: (userId) => organizationQuerier().hasRoleInOrganizationQuery(userId).organization,
})

export const organizationSearcher = (): Searcher<OrganizationSearchInput> => ({
    defaultSort: OrganizationSortBy.StarsDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [OrganizationSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [OrganizationSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [OrganizationSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [OrganizationSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [OrganizationSortBy.StarsAsc]: { stars: 'asc' },
            [OrganizationSortBy.StarsDesc]: { stars: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
                    { tags: { some: { tag: { tag: { ...insensitive } } } } },
                ]
            })
        })
    },
    customQueries(input: OrganizationSearchInput, userId: string | null | undefined): { [x: string]: any } {
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

export const organizationQuerier = () => ({
    /**
     * Query for checking if a user has a specific role in an organization
     */
    hasRoleInOrganizationQuery: (userId: string, title: string = 'Admin') => (
        { organization: { roles: { some: { title, assignees: { some: { user: { id: userId } } } } } } }
    ),
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

export const organizationMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string | null, data: OrganizationCreateInput | OrganizationUpdateInput): Promise<any> {
        // Add yourself as a member
        const members = { members: { create: { user: { connect: { id: userId ?? '' } } } } };
        // Create an Admin role assignment for yourself
        const roles = { roles: { create: { title: 'Admin', assignees: { create: { user: { connect: { id: userId ?? '' } } } } } } };
        return {
            ...members,
            ...roles,
            id: data.id,
            handle: (data as OrganizationUpdateInput).handle ?? undefined,
            isOpenToNewMembers: data.isOpenToNewMembers ?? undefined,
            isPrivate: data.isPrivate ?? undefined,
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Organization'),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: organizationTranslationCreate, update: organizationTranslationUpdate }, false),
        }
    },
    async toDBShapeUpdate(userId: string | null, data: OrganizationCreateInput | OrganizationUpdateInput): Promise<any> {
        // TODO members
        return {
            id: data.id,
            handle: (data as OrganizationUpdateInput).handle ?? null,
            isOpenToNewMembers: data.isOpenToNewMembers ?? undefined,
            isPrivate: data.isPrivate ?? undefined,
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Organization'),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: organizationTranslationCreate, update: organizationTranslationUpdate }, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0055') });
        if (createMany) {
            organizationsCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel.profanityCheck(createMany);
            createMany.forEach(input => TranslationModel.validateLineBreaks(input, ['bio'], CODE.LineBreaksBio));
            // Check if user will pass max organizations limit
            //TODO
            // const existingCount = await prisma.organization.count({ where: { members: { some: { userId: userId, role: MemberRole.Owner as any } } } });
            // if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
            //     throw new CustomError(CODE.MaxOrganizationsReached, 'Cannot create any more organizations with this account - maximum reached', { code: genErrorCode('0056') });
            // }
            // TODO handle
        }
        if (updateMany) {
            organizationsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(updateMany.map(u => u.data));
            for (const input of updateMany) {
                await WalletModel.verify(prisma).verifyHandle('Organization', input.where.id, input.data.handle);
                TranslationModel.validateLineBreaks(input.data, ['bio'], CODE.LineBreaksBio);
            }
            // Check that user is owner OR admin of each organization
            //TODO
            // const roles = userId
            //     ? await verifier.getRoles(userId, updateMany.map(input => input.where.id))
            //     : Array(updateMany.length).fill(null);
            // if (roles.some((role: any) => role !== MemberRole.Owner && role !== MemberRole.Admin)) throw new CustomError(CODE.Unauthorized);
        }
        if (deleteMany) {
            // Check that user is owner of each organization
            //TODO
            // const roles = userId
            //     ? await verifier.getRoles(userId, deleteMany)
            //     : Array(deleteMany.length).fill(null);
            // if (roles.some((role: any) => role !== MemberRole.Owner)) 
            //     throw new CustomError(CODE.Unauthorized, 'User must be owner of organization to delete', { code: genErrorCode('0057') });
        }
    },
    /**
     * Performs adds, updates, and deletes of organizations. First validates that every action is allowed.
     */
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<CUDResult<Organization>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId, input);
                // Create object
                const currCreated = await prisma.organization.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Handle members TODO
                // Find in database
                let object = await prisma.organization.findFirst({
                    where: {
                        AND: [
                            input.where,
                            { members: { some: { userId: userId ?? '' } } },
                        ]
                    }
                })
                if (!object) throw new CustomError(CODE.Unauthorized, 'User must be member of organization to update', { code: genErrorCode('0058') });
                // Update object
                const currUpdated = await prisma.organization.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.organization.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const OrganizationModel = ({
    prismaObject: (prisma: PrismaType) => prisma.organization,
    format: organizationFormatter(),
    mutate: organizationMutater,
    permissions: organizationPermissioner,
    search: organizationSearcher(),
    query: organizationQuerier,
    type: 'Organization' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================