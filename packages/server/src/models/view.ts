import { CODE, ViewFor, ViewSortBy } from "@shared/consts";
import { isObject } from '@shared/utils'
import { combineQueries, getSearchStringQueryHelper, lowercaseFirstLetter, ObjectMap, onlyValidIds, timeFrameToPrisma } from "./builder";
import { OrganizationModel, organizationQuerier } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { UserModel } from "./user";
import { StandardModel } from "./standard";
import { CustomError, genErrorCode } from "../events";
import { initializeRedis } from "../redisConn";
import { resolveProjectOrOrganizationOrRoutineOrStandardOrUser } from "../schema/resolvers";
import { User, ViewSearchInput, Count } from "../schema/types";
import { RecursivePartial, PrismaType } from "../types";
import { readManyHelper } from "./actions";
import { FormatConverter, GraphQLModelType, ModelLogic, PartialGraphQLInfo, Searcher } from "./types";

export interface View {
    __typename?: 'View';
    from: User;
    to: ViewFor;
}

export const viewFormatter = (): FormatConverter<View, any> => ({
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
    async addSupplementalFields({ objects, partial, prisma, userId }): Promise<RecursivePartial<View>[]> {
        // Query for data that view is applied to
        if (isObject(partial.to)) {
            const toTypes: GraphQLModelType[] = objects.map(o => resolveProjectOrOrganizationOrRoutineOrStandardOrUser(o.to))
            const toIds = objects.map(x => x.to?.id ?? '') as string[];
            // Group ids by types
            const toIdsByType: { [x: string]: string[] } = {};
            toTypes.forEach((type, i) => {
                if (!toIdsByType[type]) toIdsByType[type] = [];
                toIdsByType[type].push(toIds[i]);
            })
            // Query for each type
            const tos: any[] = [];
            for (const type of Object.keys(toIdsByType)) {
                const validTypes: Array<GraphQLModelType> = [
                    'Organization',
                    'Project',
                    'Routine',
                    'Standard',
                    'User',
                ];
                if (!validTypes.includes(type as GraphQLModelType)) {
                    throw new CustomError(CODE.InternalError, `View applied to unsupported type: ${type}`, { code: genErrorCode('0186') });
                }
                const model = ObjectMap[type as GraphQLModelType] as ModelLogic<any, any, any, any>;
                const paginated = await readManyHelper({
                    info: partial.to[type] as PartialGraphQLInfo,
                    input: { ids: toIdsByType[type] },
                    model,
                    prisma,
                    req: { users: [{ id: userId }] }
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
    },
})

const forMapper = {
    [ViewFor.Organization]: 'organization',
    [ViewFor.Project]: 'project',
    [ViewFor.Routine]: 'routine',
    [ViewFor.Standard]: 'standard',
    [ViewFor.User]: 'user',
}

export interface ViewInput {
    forId: string;
    title: string;
    viewFor: ViewFor;
}

export const viewSearcher = (): Searcher<ViewSearchInput> => ({
    defaultSort: ViewSortBy.LastViewedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ViewSortBy.LastViewedAsc]: { lastViewed: 'asc' },
            [ViewSortBy.LastViewedDesc]: { lastViewed: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { title: { ...insensitive } },
                    { organization: OrganizationModel.search.getSearchStringQuery(searchString, languages) },
                    { project: ProjectModel.search.getSearchStringQuery(searchString, languages) },
                    { routine: RoutineModel.search.getSearchStringQuery(searchString, languages) },
                    { standard: StandardModel.search.getSearchStringQuery(searchString, languages) },
                    { user: UserModel.search.getSearchStringQuery(searchString, languages) },
                ]
            })
        })
    },
    customQueries(input: ViewSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.lastViewedTimeFrame !== undefined ? timeFrameToPrisma('lastViewed', input.lastViewedTimeFrame) : {}),
        ])
    },
})

const viewQuerier = (prisma: PrismaType) => ({
    async getIsVieweds(
        userId: string | null,
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
const viewMutater = (prisma: PrismaType) => ({
    async view(userId: string, input: ViewInput): Promise<boolean> {
        // Define prisma type for viewed object
        const prismaFor = (prisma[forMapper[input.viewFor] as keyof PrismaType] as any);
        // Check if object being viewed on exists
        const viewFor: null | { id: string, views: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, views: true } });
        if (!viewFor)
            throw new CustomError(CODE.ErrorUnknown, 'Could not find object being viewed', { code: genErrorCode('0173') });
        // Check if view exists
        let view = await prisma.view.findFirst({
            where: {
                byId: userId,
                [`${forMapper[input.viewFor]}Id`]: input.forId
            }
        })
        // If view already existed, update view time
        if (view) {
            await prisma.view.update({
                where: { id: view.id },
                data: {
                    lastViewed: new Date(),
                    title: input.title,
                }
            })
        }
        // If view did not exist, create it
        else {
            view = await prisma.view.create({
                data: {
                    byId: userId,
                    [`${forMapper[input.viewFor]}Id`]: input.forId,
                    title: input.title,
                }
            })
        }
        // Check if a view from this user should increment the view count
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Organization:
                // Check if user is an admin or owner of the organization
                const roles = await OrganizationModel.query().hasRole(prisma, userId, [input.forId]);
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
                                    organizationQuerier().isMemberOfOrganizationQuery(userId),
                                    { user: { id: userId } },
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
                                    { createdByOrganization: organizationQuerier().isMemberOfOrganizationQuery(userId).organization },
                                    { createdByUser: { id: userId } },
                                ]
                            }
                        ]
                    }
                })
                if (object2) isOwn = true;
                break;
            case ViewFor.User:
                isOwn = userId === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Check the last time the user viewed this object
        const redisKey = `view:${userId}_${input.forId}_${input.viewFor}`
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
    },
    /**
     * Deletes views from user's view list, but does not affect view count or logs.
     */
    async deleteViews(userId: string, ids: string[]): Promise<Count> {
        return await prisma.view.deleteMany({
            where: {
                AND: [
                    { id: { in: ids } },
                    { byId: userId },
                ]
            }
        })
    },
    /**
     * Removes all of user's views, but does not affect view count or logs.
     */
    async clearViews(userId: string): Promise<Count> {
        return await prisma.view.deleteMany({
            where: { byId: userId }
        })
    },
})

export const ViewModel = ({
    prismaObject: (prisma: PrismaType) => prisma.view,
    mutate: viewMutater,
    format: viewFormatter(),
    search: viewSearcher(),
    query: viewQuerier,
    type: 'View' as GraphQLModelType,
})