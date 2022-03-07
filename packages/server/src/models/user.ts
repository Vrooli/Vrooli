import { User, UserSortBy, UserSearchInput, } from "../schema/types";
import { addJoinTablesHelper, FormatConverter, GraphQLModelType, PartialInfo, removeJoinTablesHelper, Searcher } from "./base";
import { PrismaType, RecursivePartial } from "../types";
import { StarModel } from "./star";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user' };
export const userFormatter = (): FormatConverter<User> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.User,
        'comments': GraphQLModelType.Comment,
        'resources': GraphQLModelType.Resource,
        'projects': GraphQLModelType.Project,
        'starredBy': GraphQLModelType.User,
        'reports': GraphQLModelType.Report,
        'routines': GraphQLModelType.Routine,
    },
    removeCalculatedFields: (partial) => {
        let { isStarred, ...rest } = partial;
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
    ): Promise<RecursivePartial<User>[]> {
        console.log('user addSupplementalFields', partial, objects);
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'user')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        return objects as RecursivePartial<User>[];
    },
})

export const userSearcher = (): Searcher<UserSearchInput> => ({
    defaultSort: UserSortBy.StarsDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [UserSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [UserSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [UserSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [UserSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [UserSortBy.StarsAsc]: { stars: 'asc' },
            [UserSortBy.StarsDesc]: { stars: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, bio: {...insensitive} } } },
                { username: { ...insensitive } },
            ]
        })
    },
    customQueries(input: UserSearchInput): { [x: string]: any } {
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        const resourceTypesQuery = input.resourceTypes ? { resources: { some: { usedFor: { in: input.resourceTypes } } } } : {};
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        return { ...languagesQuery, ...minStarsQuery, ...organizationIdQuery, ...projectIdQuery, ...resourceTypesQuery, ...routineIdQuery, ...reportIdQuery, ...standardIdQuery };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function UserModel(prisma: PrismaType) {
    const prismaObject = prisma.user;
    const format = userFormatter();
    const search = userSearcher();

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================