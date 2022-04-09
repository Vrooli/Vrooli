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
import { WalletModel } from "./wallet";

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
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'organization')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for role
        if (partial.role) {
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, ids)
                : Array(ids.length).fill(null);
            objects = objects.map((x, i) => ({ ...x, role: roles[i] })) as any;
        }
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
        // Query member data for each ID
        const roleArray = await prisma.organization_users.findMany({
            where: { organization: { id: { in: ids } }, user: { id: userId } },
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
        // If creating new, add yourself as member
        let members = {};
        if (!(data as OrganizationUpdateInput).id) {
            members = { members: { create: { userId, role: MemberRole.Owner as any } } }
        }
        return {
            ...members,
            handle: (data as OrganizationUpdateInput).handle ?? null,
            isOpenToNewMembers: data.isOpenToNewMembers,
            resourceLists: await ResourceListModel(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel(prisma).relationshipBuilder(userId, data),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: organizationTranslationCreate, update: organizationTranslationUpdate }, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => organizationCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            createMany.forEach(input => TranslationModel().validateLineBreaks(input, ['bio'], CODE.LineBreaksBio));
            // Check if user will pass max organizations limit
            const existingCount = await prisma.organization.count({ where: { members: { some: { userId: userId ?? '', role: MemberRole.Owner as any } } } });
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.MaxOrganizationsReached);
            }
        }
        if (updateMany) {
            for (const input of updateMany) {
                await WalletModel(prisma).verifyHandle(GraphQLModelType.Organization, input.where.id, input.data.handle);
                organizationUpdate.validateSync(input.data, { abortEarly: false });
                TranslationModel().profanityCheck(input.data);
                TranslationModel().validateLineBreaks(input.data, ['bio'], CODE.LineBreaksBio);
            }
            // Check that user is owner OR admin of each organization
            const roles = userId
                ? await verifier.getRoles(userId, updateMany.map(input => input.where.id))
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
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                // Create object
                const currCreated = await prisma.organization.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Handle members TODO
                const temp = await this.toDBShape(userId, input.data)
                console.log('organization update a', JSON.stringify(input));
                console.log('organization update b', JSON.stringify(temp));
                // Find in database
                let object = await prisma.organization.findFirst({
                    where: {
                        AND: [
                            input.where,
                            { members: { some: { userId: userId ?? '' } } },
                        ]
                    }
                })
                if (!object) throw new CustomError(CODE.Unauthorized);
                // Update object
                const currUpdated = await prisma.organization.update({
                    where: input.where,
                    data: await this.toDBShape(userId, input.data),
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