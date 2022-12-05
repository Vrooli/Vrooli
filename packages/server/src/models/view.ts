import { ViewFor, ViewSortBy } from "@shared/consts";
import { isObject } from '@shared/utils'
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { UserModel } from "./user";
import { StandardModel } from "./standard";
import { CustomError } from "../events";
import { initializeRedis } from "../redisConn";
import { resolveUnion } from "../endpoints/resolvers";
import { User, ViewSearchInput, Count, SessionUser } from "../endpoints/types";
import { RecursivePartial, PrismaType } from "../types";
import { readManyHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType, Searcher } from "./types";
import { Prisma } from "@prisma/client";
import { PartialGraphQLInfo } from "../builders/types";
import { combineQueries, lowercaseFirstLetter, onlyValidIds, padSelect, timeFrameToPrisma } from "../builders";
import { getLabels } from "../getters";

interface View {
    __typename?: 'View';
    from: User;
    to: ViewFor;
}

const formatter = (): Formatter<View, 'to'> => ({
    relationshipMap: {
        __typename: 'View',
        from: 'User',
        to: {
            Organization: 'Organization',
            Project: 'Project',
            Routine: 'Routine',
            Standard: 'Standard',
            User: 'User',
        }
    },
    supplemental: {
        graphqlFields: ['to'],
        toGraphQL: ({ languages, objects, partial, prisma, userData }) => [
            ['to', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Query for data that view is applied to
                if (isObject(partial.to)) {
                    const toTypes: GraphQLModelType[] = objects.map(o => resolveUnion(o.to))
                    const toIds = objects.map(x => x.to?.id ?? '') as string[];
                    // Group ids by types
                    const toIdsByType: { [x: string]: string[] } = {};
                    toTypes.forEach((type, i) => {
                        if (!toIdsByType[type]) toIdsByType[type] = [];
                        toIdsByType[type].push(toIds[i]);
                    })
                    // Query for each type
                    const tos: any[] = [];
                    for (const objectType of Object.keys(toIdsByType)) {
                        const validTypes: Array<GraphQLModelType> = [
                            'Organization',
                            'Project',
                            'Routine',
                            'Standard',
                            'User',
                        ];
                        if (!validTypes.includes(objectType as GraphQLModelType)) {
                            throw new CustomError('0186', 'InternalError', languages, { objectType });
                        }
                        const paginated = await readManyHelper({
                            info: partial.to[objectType] as PartialGraphQLInfo,
                            input: { ids: toIdsByType[objectType] },
                            objectType: objectType as GraphQLModelType,
                            prisma,
                            req: { languages, users: [userData] }
                        })
                        tos.push(...paginated.edges.map(x => x.node));
                    }
                    // Apply each "to" to the "to" property of each object
                    for (const object of objects) {
                        // Find the correct "to", using object.to.id
                        const to = tos.find(x => x.id === object.to.id);
                        object.to = to;
                    }
                }
                return objects as RecursivePartial<View>[];
            }],
        ],
    },
})

const forMapper: { [key in ViewFor]: keyof PrismaType } = {
    Organization: 'organization',
    Project: 'project_version',
    Routine: 'routine_version',
    Standard: 'standard_version',
    User: 'user',
}

interface ViewInput {
    forId: string;
    viewFor: ViewFor;
}

const searcher = (): Searcher<
    ViewSearchInput,
    ViewSortBy,
    Prisma.viewOrderByWithRelationInput,
    Prisma.viewWhereInput
> => ({
    defaultSort: ViewSortBy.LastViewedDesc,
    sortMap: {
        LastViewedAsc: { lastViewed: 'asc' },
        LastViewedDesc: { lastViewed: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages, searchString }) => ({
        OR: [
            { title: { ...insensitive } },
            { organization: OrganizationModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { project: ProjectModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { routine: RoutineModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { standard: StandardModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { user: UserModel.search.searchStringQuery({ insensitive, languages, searchString }) },
        ]
    }),
    customQueries(input) {
        return combineQueries([
            (input.lastViewedTimeFrame !== undefined ? timeFrameToPrisma('lastViewed', input.lastViewedTimeFrame) : {}),
        ])
    },
})

const querier = () => ({
    async getIsVieweds(
        prisma: PrismaType,
        userId: string | null | undefined,
        ids: string[],
        viewFor: keyof typeof ViewFor
    ): Promise<Array<boolean | null>> {
        // Create result array that is the same length as ids
        const result = new Array(ids.length).fill(null);
        // If userId not provided, return result
        if (!userId) return result;
        // Filter out nulls and undefineds from ids
        const idsFiltered = onlyValidIds(ids);
        const fieldName = `${viewFor.toLowerCase()}Id`;
        const isViewedArray = await prisma.view.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
        // Replace the nulls in the result array with true if viewed
        for (let i = 0; i < ids.length; i++) {
            // Try to find this id in the isViewed array
            result[i] = Boolean(isViewedArray.find((view: any) => view[fieldName] === ids[i]));
        }
        return result;
    },
})

/**
 * Marks objects as viewed. If view exists, updates last viewed time.
 * A user may view their own objects, but it does not count towards its view count.
 * @returns True if view updated correctly
 */
const view = async (prisma: PrismaType, userData: SessionUser, input: ViewInput): Promise<boolean> => {
    // Define prisma type for viewed object
    const prismaFor = (prisma[forMapper[input.viewFor] as keyof PrismaType] as any);
    // Check if object being viewed on exists
    const viewFor: null | { id: string, views: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, views: true } });
    if (!viewFor)
        throw new CustomError('0173', 'NotFound', userData.languages);
    // Check if view exists
    let view = await prisma.view.findFirst({
        where: {
            byId: userData.id,
            [`${forMapper[input.viewFor]}Id`]: input.forId
        }
    })
    // If view already existed, update view time
    if (view) {
        await prisma.view.update({
            where: { id: view.id },
            data: {
                lastViewed: new Date(),
            }
        })
    }
    // If view did not exist, create it
    else {
        const labels = await getLabels([{ id: input.forId, languages: userData.languages }], input.viewFor, prisma, userData.languages, 'view');
        view = await prisma.view.create({
            data: {
                byId: userData.id,
                [`${forMapper[input.viewFor]}Id`]: input.forId,
                title: labels[0],
            }
        })
    }
    // Check if a view from this user should increment the view count
    let isOwn = false;
    switch (input.viewFor) {
        case ViewFor.Organization:
            // Check if user is an admin or owner of the organization
            const roles = await OrganizationModel.query.hasRole(prisma, userData.id, [input.forId]);
            isOwn = Boolean(roles[0]);
            break;
        case ViewFor.Project:
        case ViewFor.Routine:
            // Check if project/routine is owned by this user or by an organization they are a member of
            const object = await (prisma[lowercaseFirstLetter(input.viewFor) as 'project' | 'routine'] as any).findFirst({
                where: {
                    AND: [
                        { id: input.forId },
                        {
                            OR: [
                                OrganizationModel.query.isMemberOfOrganizationQuery(userData.id),
                                { user: { id: userData.id } },
                            ]
                        }
                    ]
                }
            })
            if (object) isOwn = true;
            break;
        case ViewFor.Standard:
            // Check if standard is owned by this user or by an organization they are a member of
            const object2 = await prisma.standard.findFirst({
                where: {
                    AND: [
                        { id: input.forId },
                        {
                            OR: [
                                { ownedByOrganization: OrganizationModel.query.isMemberOfOrganizationQuery(userData.id).organization },
                                { ownedByUser: { id: userData.id } },
                            ]
                        }
                    ]
                }
            })
            if (object2) isOwn = true;
            break;
        case ViewFor.User:
            isOwn = userData.id === input.forId;
            break;
    }
    // If user is owner, don't do anything else
    if (isOwn) return true;
    // Check the last time the user viewed this object
    const redisKey = `view:${userData.id}_${input.forId}_${input.viewFor}`
    const client = await initializeRedis();
    const lastViewed = await client.get(redisKey);
    // If object viewed more than 1 hour ago, update view count
    if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
        await prismaFor.update({
            where: { id: input.forId },
            data: { views: viewFor.views + 1 }
        })
    }
    // Update last viewed time
    await client.set(redisKey, new Date().toISOString());
    return true;
}

/**
 * Deletes views from user's view list, but does not affect view count or logs.
 */
const deleteViews = async (prisma: PrismaType, userId: string, ids: string[]): Promise<Count> => {
    return await prisma.view.deleteMany({
        where: {
            AND: [
                { id: { in: ids } },
                { byId: userId },
            ]
        }
    })
}

/**
 * Removes all of user's views, but does not affect view count or logs.
 */
const clearViews = async (prisma: PrismaType, userId: string): Promise<Count> => {
    return await prisma.view.deleteMany({
        where: { byId: userId }
    })
}

const displayer = (): Displayer<
    Prisma.viewSelect,
    Prisma.viewGetPayload<{ select: { [K in keyof Required<Prisma.viewSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        // api: padSelect(ApiModel.display.select),
        organization: padSelect(OrganizationModel.display.select),
        // post: padSelect(PostModel.display.select),
        project: padSelect(ProjectModel.display.select),
        routine: padSelect(RoutineModel.display.select),
        // smartContract: padSelect(SmartContractModel.display.select),
        standard: padSelect(StandardModel.display.select),
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => {
        // if (select.api) return ApiModel.display.label(select.api as any, languages);
        if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
        if (select.project) return ProjectModel.display.label(select.project as any, languages);
        if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
        // if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
        if (select.standard) return StandardModel.display.label(select.standard as any, languages);
        if (select.user) return UserModel.display.label(select.user as any, languages);
        return '';
    }
})

export const ViewModel = ({
    delegate: (prisma: PrismaType) => prisma.view,
    display: displayer(),
    format: formatter(),
    search: searcher(),
    query: querier(),
    type: 'View' as GraphQLModelType,
    view,
    deleteViews,
    clearViews,
})