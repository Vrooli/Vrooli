import { addCountFieldsHelper, addJoinTablesHelper, combineQueries, getSearchStringQueryHelper, removeCountFieldsHelper, removeJoinTablesHelper } from "./builder";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { UserSortBy, ResourceListUsedFor } from "@shared/consts";
import { User, UserSearchInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, GraphQLModelType, Validator } from "./types";
import { Prisma } from "@prisma/client";

const joinMapper = { starredBy: 'user' };
const countMapper = { reportsCount: 'reports' };
type SupplementalFields = 'isStarred' | 'isViewed';
export const userFormatter = (): FormatConverter<User, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'User',
        comments: 'Comment',
        resourceLists: 'ResourceList',
        projects: 'Project',
        starredBy: 'User',
        reports: 'Report',
        routines: 'Routine',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isViewed'],
        toGraphQL: ({ ids, prisma, userId }) => [
            ['isStarred', async () => await StarModel.query(prisma).getIsStarreds(userId, ids, 'User')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'User')],
        ],
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
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
                    { name: { ...insensitive } },
                    { handle: { ...insensitive } },
                ]
            })
        })
    },
    customQueries(input: UserSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.organizationId !== undefined ? { organizations: { some: { organizationId: input.organizationId } } } : {}),
            (input.projectId !== undefined ? { projects: { some: { projectId: input.projectId } } } : {}),
            (input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            (input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.standardId !== undefined ? { standards: { some: { id: input.standardId } } } : {}),
        ])
    },
})

export const userValidator = (): Validator<
    any,
    any,
    User,
    Prisma.userGetPayload<{ select: { [K in keyof Required<Prisma.userSelect>]: true } }>,
    any,
    Prisma.userSelect,
    Prisma.userWhereInput
> => ({
    validateMap: {
        __typename: 'User',
        projects: 'Project',
        resourceLists: 'ResourceList',
        reports: 'Report',
        routines: 'Routine',
    },
    permissionsSelect: () => ({ id: true, isPrivate: true }),
    permissionResolvers: (data, userId) => fdasfds as any,
    ownerOrMemberWhere: (userId) => ({ id: userId }),
    isAdmin: (data, userId) => data.id === userId,
    isPublic: (data) => data.isPrivate === false,
    profanityFields: ['name', 'handle'],
})

export const UserModel = ({
    prismaObject: (prisma: PrismaType) => prisma.user,
    format: userFormatter(),
    search: userSearcher(),
    type: 'User' as GraphQLModelType,
    validate: userValidator(),
})