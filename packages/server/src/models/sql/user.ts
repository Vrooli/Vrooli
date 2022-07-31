import { User, UserSortBy, UserSearchInput, ResourceListUsedFor, } from "../../schema/types";
import { addCountFieldsHelper, addJoinTablesHelper, FormatConverter, GraphQLModelType, ModelLogic, PartialGraphQLInfo, removeCountFieldsHelper, removeJoinTablesHelper, Searcher } from "./base";
import { PrismaType, RecursivePartial } from "../../types";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { omit } from "@local/shared";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user' };
const countMapper = { reportsCount: 'reports' };
const calculatedFields = ['isStarred'];
export const userFormatter = (): FormatConverter<User> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.User,
        'comments': GraphQLModelType.Comment,
        'resourceLists': GraphQLModelType.ResourceList,
        'projects': GraphQLModelType.Project,
        'starredBy': GraphQLModelType.User,
        'reports': GraphQLModelType.Report,
        'routines': GraphQLModelType.Routine,
    },
    removeCalculatedFields: (partial) => {
        return omit(partial, calculatedFields);
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
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialGraphQLInfo,
    ): Promise<RecursivePartial<User>[]> {
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel.query(prisma).getIsStarreds(userId, ids, GraphQLModelType.User)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isViewed
        if (partial.isViewed) {
            const isViewedArray = userId
                ? await ViewModel.query(prisma).getIsVieweds(userId, ids, GraphQLModelType.User)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isViewed: isViewedArray[i] }));
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
                { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
                { name: { ...insensitive } },
                { handle: { ...insensitive } },
            ]
        })
    },
    customQueries(input: UserSearchInput): { [x: string]: any } {
        return {
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            ...(input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            ...(input.organizationId !== undefined ? { organizations: { some: { organizationId: input.organizationId } } } : {}),
            ...(input.projectId !== undefined ? { projects: { some: { projectId: input.projectId } } } : {}),
            ...(input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            ...(input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            ...(input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            ...(input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.standardId !== undefined ? { standards: { some: { id: input.standardId } } } : {}),
        }
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const UserModel = ({
    prismaObject: (prisma: PrismaType) => prisma.user,
    format: userFormatter(),
    search: userSearcher(),
})

//==============================================================
/* #endregion Model */
//==============================================================