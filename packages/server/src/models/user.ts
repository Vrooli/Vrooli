import { User, UserSortBy, UserSearchInput, UserCountInput, } from "../schema/types";
import { addJoinTablesHelper, counter, FormatConverter, infoToPartialSelect, InfoType, ModelTypes, removeJoinTablesHelper, Searcher } from "./base";
import { PrismaType, RecursivePartial } from "../types";
import { StarModel } from "./star";

export const userDBFields = ['id', 'created_at', 'updated_at', 'bio', 'username', 'theme', 'numExports', 'lastExport', 'status', 'stars'];

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user' };
export const userFormatter = (): FormatConverter<User> => ({
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
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<User>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
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
    defaultSort: UserSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [UserSortBy.AlphabeticalAsc]: { username: 'asc' },
            [UserSortBy.AlphabeticalDesc]: { username: 'desc' },
            [UserSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [UserSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [UserSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [UserSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [UserSortBy.StarsAsc]: { stars: 'asc' },
            [UserSortBy.StarsDesc]: { stars: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { username: { ...insensitive } },
            ]
        })
    },
    customQueries(input: UserSearchInput): { [x: string]: any } {
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        return { ...organizationIdQuery, ...projectIdQuery, ...routineIdQuery, ...reportIdQuery, ...standardIdQuery };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function UserModel(prisma: PrismaType) {
    const model = ModelTypes.User;
    const prismaObject = prisma.user;
    const format = userFormatter();
    const search = userSearcher();

    return {
        model,
        prismaObject,
        ...format,
        ...search,
        ...counter<UserCountInput>(model, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================