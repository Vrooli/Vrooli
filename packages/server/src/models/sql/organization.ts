import { PrismaType, RecursivePartial } from "../../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Count, ResourceListUsedFor, OrganizationPermission } from "../../schema/types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, removeJoinTablesHelper, Searcher, selectHelper, modelToGraphQL, ValidateMutationsInput, addCountFieldsHelper, removeCountFieldsHelper, addSupplementalFieldsHelper, Permissioner } from "./base";
import { CustomError } from "../../error";
import { CODE, omit, organizationsCreate, organizationsUpdate, organizationTranslationCreate, organizationTranslationUpdate } from "@local/shared";
import { organization_users, role } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { WalletModel } from "./wallet";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";

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
        'projectsCreated': 'Project',
        'reports': 'Report',
        'resourceLists': 'ResourceList',
        'routines': 'Routine',
        'routersCreated': 'Routine',
        'standards': 'Standard',
        'starredBy': 'User',
        'tags': 'Tag',
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
    addCountFields: (partial) => {
        return addCountFieldsHelper(partial, countMapper);
    },
    removeCountFields: (data) => {
        return removeCountFieldsHelper(data, countMapper);
    },
    removeSupplementalFields: (partial) => {
        return omit(partial, supplementalFields);
    },
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Organization>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Organization')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Organization')],
                ['permissionsOrganization', async () => await OrganizationModel.permissions(prisma).get({ objects, permissions, userId })],
            ]
        });
    }
    // // Query for role
    // if (partial.role) {
    //     const roles = userId
    //         ? await OrganizationModel.query(prisma).getRoles(userId, ids)
    //         : Array(ids.length).fill(null);
    //     objects = objects.map((x, i) => ({ ...x, role: roles[i] })) as any;
    // }
})

export const organizationPermissioner = (prisma: PrismaType): Permissioner<OrganizationPermission, OrganizationSearchInput> => ({
    async get({
        objects,
        permissions,
        userId,
    }) {
        //TODO
        return objects.map((o) => ({
            canAddMembers: true,
            canDelete: true,
            canEdit: true,
            canReport: true,
            canStar: true,
            canView: true,
            isMember: true,
        }));
    },
    async canSearch({
        input,
        userId
    }) {
        //TODO
        return 'full';
    }
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
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
                { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    },
    customQueries(input: OrganizationSearchInput): { [x: string]: any } {
        return {
            ...(input.isOpenToNewMembers !== undefined ? { isOpenToNewMembers: true } : {}),
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            ...(input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            ...(input.projectId !== undefined ? { projects: { some: { projectId: input.projectId } } } : {}),
            ...(input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            ...(input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            ...(input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            //TODO
            // ...(input.userId !== undefined ? { members: { some: { userId: input.userId, role: { in: [MemberRole.Admin, MemberRole.Owner] } } } } : {}),
            ...(input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.standardId !== undefined ? { standards: { some: { id: input.standardId } } } : {}),
            ...(input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        }
    },
})

export const organizationQuerier = (prisma: PrismaType) => ({
    /**
     * Finds the roles a users has for a list of organizations
     * @param userId The user's id
     * @param organizationIds The list of organization ids
     * @returns An array in the same order as the ids, with either a role or undefined
     */
    async getRoles(userId: string, organizationIds: (string | null | undefined)[]): Promise<Array<any | undefined>> {
        //TODO
        return Array(organizationIds.length).fill(undefined);
        // if (organizationIds.length === 0) return [];
        // // Take out nulls
        // const idsToQuery = organizationIds.filter(x => x) as string[];
        // // Query member data for each ID
        // const memberData = await prisma.organization_users.findMany({
        //     where: { organization: { id: { in: idsToQuery } }, user: { id: userId } },
        //     select: { organizationId: true, role: true }
        // });
        // // Create an array of the same length as the input, with the member data or undefined
        // return organizationIds.map(id => memberData.find(({ organizationId }) => organizationId === id)).map((o) => o?.role) as Array<MemberRole | undefined>;
    },
    /**
     * Determines if a user has a role of a list of organizations
     * @param userId The user's ID
     * @param organizationIds The list of organization IDs
     * @param title The name of the role
     * @returns Array in the same order as the ids, with either admin/owner role data or undefined
     */
    async hasRole(userId: string, organizationIds: (string | null | undefined)[], title: string = 'Admin'): Promise<Array<role | undefined>> {
        if (organizationIds.length === 0) return [];
        // Take out nulls
        const idsToQuery = organizationIds.filter(x => x) as string[];
        // Query roles data for each organization ID
        const roles = await prisma.role.findMany({
            where: {
                title,
                organization: { id: { in: idsToQuery } },
                assignees: { some: { id: userId } }
            }
        })
        // Create an array of the same length as the input, with the role data or undefined
        return organizationIds.map(id => roles.find(({ organizationId }) => organizationId === id));
    },
})

export const organizationMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string | null, data: OrganizationCreateInput | OrganizationUpdateInput): Promise<any> {
        // Add yourself as member
        //TODO
        // let members = { members: { create: { userId, role: MemberRole.Owner as any } } };
        return {
            // ...members,
            id: data.id,
            handle: (data as OrganizationUpdateInput).handle ?? null,
            isOpenToNewMembers: data.isOpenToNewMembers,
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
            isOpenToNewMembers: data.isOpenToNewMembers,
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
})

//==============================================================
/* #endregion Model */
//==============================================================