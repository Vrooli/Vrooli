import { StarModel } from "./star";
import { ViewModel } from "./view";
import { UserSortBy, ResourceListUsedFor } from "@shared/consts";
import { ProfileUpdateInput, User, UserSearchInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Displayer, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { profilesUpdate } from "@shared/validation";
import { combineQueries } from "../builders";
import { translationRelationshipBuilder } from "../utils";

type SupplementalFields = 'isStarred' | 'isViewed';
const formatter = (): Formatter<User, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'User',
        comments: 'Comment',
        resourceLists: 'ResourceList',
        projects: 'Project',
        starredBy: 'User',
        reports: 'Report',
        routines: 'Routine',
    },
    joinMap: { starredBy: 'user' },
    countMap: { reportsCount: 'reports' },
    supplemental: {
        graphqlFields: ['isStarred', 'isViewed'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'User')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'User')],
        ],
    },
})

export const searcher = (): Searcher<
    UserSearchInput,
    UserSortBy,
    Prisma.userOrderByWithRelationInput,
    Prisma.userWhereInput
> => ({
    defaultSort: UserSortBy.StarsDesc,
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
            { name: { ...insensitive } },
            { handle: { ...insensitive } },
        ]
    }),
    customQueries(input) {
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

const validator = (): Validator<
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
        reportsCreated: 'Report',
        routines: 'Routine',
        // userSchedules: 'UserSchedule',
    },
    isTransferable: false,
    permissionsSelect: () => ({
        id: true,
        isPrivate: true,
        languages: { select: { language: true } },
    }),
    permissionResolvers: () => [],
    ownerOrMemberWhere: (userId) => ({ id: userId }),
    owner: (data) => ({ User: data }),
    isDeleted: () => false,
    isPublic: (data) => data.isPrivate === false,
    profanityFields: ['name', 'handle'],
    // createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
})

const mutater = (): Mutater<
    User,
    false,
    { graphql: ProfileUpdateInput, db: Prisma.userUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        update: async ({ data, prisma, userData }) => {
            return {
                handle: data.handle,
                name: data.name ?? undefined,
                theme: data.theme ?? undefined,
                // // hiddenTags: await TagHiddenModel.mutate(prisma).relationshipBuilder!(userData.id, input, false),
                // starred: {
                //     create: starredCreate,
                //     delete: starredDelete,
                // }, TODO
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        }
    },
    yup: { update: profilesUpdate },
})

const displayer = (): Displayer => ({
    labels: async (prisma, objects) => {
        const nameData = await prisma.user.findMany({
            where: { id: { in: objects.map((o) => o.id) } },
            select: { id: true, name: true }
        })
        return objects.map(o => {
            const name = nameData.find(n => n.id === o.id)?.name;
            return name ?? '';
        })
    }
})

export const UserModel = ({
    delegate: (prisma: PrismaType) => prisma.user,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'User' as GraphQLModelType,
    validate: validator(),
})