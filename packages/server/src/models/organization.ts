import { PrismaType, RecursivePartial } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Count, ResourceListUsedFor } from "../schema/types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, removeJoinTablesHelper, Searcher, selectHelper, modelToGraphQL, ValidateMutationsInput, GraphQLModelType, PartialInfo } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationCreate, organizationTranslationCreate, organizationTranslationUpdate, organizationUpdate } from "@local/shared";
import { organization_users } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user', tags: 'tag' };
export const organizationFormatter = (): FormatConverter<Organization> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Organization,
        'comments': GraphQLModelType.Comment,
        'members': GraphQLModelType.Member,
        'projects': GraphQLModelType.Project,
        'projectsCreated': GraphQLModelType.Project,
        'reports': GraphQLModelType.Report,
        'resourceLists': GraphQLModelType.ResourceList,
        'routines': GraphQLModelType.Routine,
        'routersCreated': GraphQLModelType.Routine,
        'standards': GraphQLModelType.Standard,
        'starredBy': GraphQLModelType.User,
        'tags': GraphQLModelType.Tag,
    },
    removeCalculatedFields: (partial) => {
        let { isStarred, role, ...rest } = partial;
        return rest;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        console.log('in organization removeJoinTables', data);
        return removeJoinTablesHelper(data, joinMapper);
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialInfo,
    ): Promise<RecursivePartial<Organization>[]> {
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        console.log('in organization addSupplementalFields', ids, userId, partial);
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'organization')
                : Array(ids.length).fill(false);
            console.log('filling in isStarred', isStarredArray, objects);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
            console.log('objectssssss FILLED', objects);
        }
        // Query for role
        if (partial.role) {
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, ids)
                : Array(ids.length).fill(null);
            objects = objects.map((x, i) => ({ ...x, role: roles[i] })) as any;
        }
        console.log('organization supplemental fields complete', objects);
        return objects as RecursivePartial<Organization>[];
    },
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
        const isOpenToNewMembersQuery = input.isOpenToNewMembers ? { isOpenToNewMembers: true } : {};
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        const resourceListsQuery = input.resourceLists ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {};
        const resourceTypesQuery = input.resourceTypes ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {};
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { userId: input.userId, role: { in: [MemberRole.Admin, MemberRole.Owner] } } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        const tagsQuery = input.tags ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {};
        return { ...isOpenToNewMembersQuery, ...languagesQuery, ...minStarsQuery, ...projectIdQuery, ...resourceListsQuery, ...resourceTypesQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery, ...tagsQuery };
    },
})

export const organizationVerifier = (prisma: PrismaType) => ({
    async getRoles(userId: string, ids: string[]): Promise<Array<MemberRole | undefined>> {
        console.log('getroles start', userId, ids);
        // Query member data for each ID
        const roleArray = await prisma.organization_users.findMany({
            where: { organization: { id: { in: ids } }, userId },
            select: { organizationId: true, role: true }
        });
        return ids.map(id => {
            const role = roleArray.find(({ organizationId }) => organizationId === id);
            return role?.role as MemberRole | undefined;
        });
    },
    async isOwnerOrAdmin(userId: string, organizationId: string): Promise<[boolean, organization_users | null]> {
        const memberData = await prisma.organization_users.findFirst({
            where: {
                organization: { id: organizationId },
                user: { id: userId },
                role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
            }
        });
        return [Boolean(memberData), memberData];
    },
})

export const organizationMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShape(userId: string | null, data: OrganizationCreateInput | OrganizationUpdateInput): Promise<any> {
        return {
            id: (data as OrganizationUpdateInput)?.id ?? undefined,
            isOpenToNewMembers: data.isOpenToNewMembers,
            resourceLists: await ResourceListModel(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel(prisma).relationshipBuilder(userId, data, false),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: organizationTranslationCreate, update: organizationTranslationUpdate }, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<void> {
        console.log('in organization validatemutations')
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            console.log('creating many validating', createMany);
            createMany.forEach(input => organizationCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            createMany.forEach(input => TranslationModel().validateLineBreaks(input, ['bio'], CODE.LineBreaksBio));
            console.log('passed create many validations');
            // Check if user will pass max organizations limit
            const existingCount = await prisma.organization.count({ where: { members: { some: { userId: userId ?? '', role: MemberRole.Owner as any } } } });
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.MaxOrganizationsReached);
            }
        }
        if (updateMany) {
            updateMany.forEach(input => organizationUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => TranslationModel().profanityCheck(input));
            updateMany.forEach(input => TranslationModel().validateLineBreaks(input, ['bio'], CODE.LineBreaksBio));
            // Check that user is owner OR admin of each organization
            const roles = userId
                ? await verifier.getRoles(userId, updateMany.map(input => input.id))
                : Array(updateMany.length).fill(null);
            if (roles.some((role: any) => role !== MemberRole.Owner && role !== MemberRole.Admin)) throw new CustomError(CODE.Unauthorized);
        }
        if (deleteMany) {
            // Check that user is owner of each organization
            const roles = userId
                ? await verifier.getRoles(userId, deleteMany)
                : Array(deleteMany.length).fill(null);
            if (roles.some((role: any) => role !== MemberRole.Owner)) throw new CustomError(CODE.Unauthorized);
        }
    },
    /**
     * Performs adds, updates, and deletes of organizations. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<CUDResult<Organization>> {
        console.log('in organization cud start', partial, userId, createMany);
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                console.log('organization create input loop db shape', data);
                // Create object
                const currCreated = await prisma.organization.create({ data, ...selectHelper(partial) });
                console.log('organization create input loop currCreated', currCreated);
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                console.log('organization create input loop tographql', converted)
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                // Handle members TODO
                // Find in database
                let object = await prisma.organization.findFirst({
                    where: {
                        AND: [
                            { id: input.id },
                            { members: { some: { id: userId ?? '' } } },
                        ]
                    }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Update object
                const currUpdated = await prisma.organization.update({
                    where: { id: object.id },
                    data,
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
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

export function OrganizationModel(prisma: PrismaType) {
    const prismaObject = prisma.organization;
    const format = organizationFormatter();
    const search = organizationSearcher();
    const verify = organizationVerifier(prisma);
    const mutate = organizationMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================