import { PrismaType, RecursivePartial } from "../types";
import { Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Count } from "../schema/types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, infoToPartialSelect, InfoType, removeJoinTablesHelper, Searcher, selectHelper, modelToGraphQL, ValidateMutationsInput } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationCreate, organizationUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { ResourceModel } from "./resource";
import { organization_users } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";

export const organizationDBFields = ['id', 'created_at', 'updated_at', 'bio', 'name', 'openToNewMembers', 'stars'];

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user', tags: 'tag' };
export const organizationFormatter = (): FormatConverter<Organization> => ({
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
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Organization>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
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
        console.log('organization supplemental fields complete', objects);
        return objects as RecursivePartial<Organization>[];
    },
})

export const organizationSearcher = (): Searcher<OrganizationSearchInput> => ({
    defaultSort: OrganizationSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [OrganizationSortBy.AlphabeticalAsc]: { name: 'asc' },
            [OrganizationSortBy.AlphabeticalDesc]: { name: 'desc' },
            [OrganizationSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [OrganizationSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [OrganizationSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [OrganizationSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [OrganizationSortBy.StarsAsc]: { stars: 'asc' },
            [OrganizationSortBy.StarsDesc]: { stars: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { bio: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    },
    customQueries(input: OrganizationSearchInput): { [x: string]: any } {
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { id: input.userId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        return { ...projectIdQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery }
    },
})

export const organizationVerifier = (prisma: PrismaType) => ({
    async getRoles(userId: string, ids: string[]): Promise<Array<MemberRole | undefined>> {
        // Query member data for each ID
        const roleArray = await prisma.organization_users.findMany({
            where: { organizationId: { in: ids }, user: { id: userId } },
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
    profanityCheck(data: OrganizationCreateInput | OrganizationUpdateInput): void {
        if (hasProfanity(data.name, data.bio)) throw new CustomError(CODE.BannedWord);
    },
})

export const organizationMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShape(userId: string | null, data: OrganizationCreateInput | OrganizationUpdateInput): Promise<any> {
        return {
            name: data.name,
            bio: data.bio,
            resources: ResourceModel(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel(prisma).relationshipBuilder(userId, data, false),
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => organizationCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check if user will pass max organizations limit
            const existingCount = await prisma.organization.count({ where: { members: { some: { userId: userId ?? '', role: MemberRole.Owner as any } } } });
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.ErrorUnknown, 'To prevent spam, user cannot own more than 100 organizations. If you think this is a mistake, please contact us');
            }
        }
        if (updateMany) {
            updateMany.forEach(input => organizationUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
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
    async cud({ info, userId, createMany, updateMany, deleteMany }: CUDInput<OrganizationCreateInput, OrganizationUpdateInput>): Promise<CUDResult<Organization>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                // Create object
                const currCreated = await prisma.organization.create({ data, ...selectHelper(info) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, info);
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
                    ...selectHelper(info)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, info);
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