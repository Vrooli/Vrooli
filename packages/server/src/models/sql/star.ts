import { CODE, StarFor } from "@shared/consts";
import { isObject } from '@shared/utils'; 
import { CustomError } from "../../error";
import { LogType, Star, StarInput, StarSearchInput, StarSortBy } from "../../schema/types";
import { PrismaType, RecursivePartial } from "../../types";
import { combineQueries, FormatConverter, getSearchStringQueryHelper, ModelLogic, ObjectMap, onlyValidIds, readManyHelper, Searcher } from "./base";
import { genErrorCode, logger, LogLevel } from "../../logger";
import { Log } from "../../models/nosql";
import { resolveStarTo } from "../../schema/resolvers";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { CommentModel } from "./comment";
import { GraphQLModelType } from ".";

//==============================================================
/* #region Custom Components */
//==============================================================

export const starFormatter = (): FormatConverter<Star, any> => ({
    relationshipMap: {
        '__typename': 'Star',
        'from': 'User',
        'to': {
            'Comment': 'Comment',
            'Organization': 'Organization',
            'Project': 'Project',
            'Routine': 'Routine',
            'Standard': 'Standard',
            'Tag': 'Tag',
            'User': 'User',
        }
    },
    unionMap: {
        'to': {
            'Comment': 'comment',
            'Organization': 'organization',
            'Project': 'project',
            'Routine': 'routine',
            'Standard': 'standard',
            'Tag': 'tag',
            'User': 'user',
        },
    },
    async addSupplementalFields({ objects, partial, prisma, userId }): Promise<RecursivePartial<Star>[]> {
        // Query for data that star is applied to
        if (isObject(partial.to)) {
            const toTypes: GraphQLModelType[] = objects.map(o => resolveStarTo(o.to)).filter(t => t);
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
                const validTypes: Array<keyof typeof GraphQLModelType> = [
                    'Comment',
                    'Organization',
                    'Project',
                    'Routine',
                    'Standard',
                    'Tag',
                    'User',
                ];
                if (!validTypes.includes(type as keyof typeof GraphQLModelType)) {
                    throw new CustomError(CODE.InternalError, `View applied to unsupported type: ${type}`, { code: genErrorCode('0185') });
                }
                const model: ModelLogic<any, any, any> = ObjectMap[type as keyof typeof GraphQLModelType] as ModelLogic<any, any, any>;
                const paginated = await readManyHelper({
                    info: partial.to[type],
                    input: { ids: toIdsByType[type] },
                    model,
                    prisma,
                    userId,
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
        return objects;
    },
})

export const starSearcher = (): Searcher<StarSearchInput> => ({
    defaultSort: StarSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [StarSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [StarSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({ searchString,
            resolver: () => ({ 
                OR: [
                    { organization: OrganizationModel.search.getSearchStringQuery(searchString, languages) },
                    { project: ProjectModel.search.getSearchStringQuery(searchString, languages) },
                    { routine: RoutineModel.search.getSearchStringQuery(searchString, languages) },
                    { standard: StandardModel.search.getSearchStringQuery(searchString, languages) },
                    { tag: TagModel.search.getSearchStringQuery(searchString, languages) },
                    { comment: CommentModel.search.getSearchStringQuery(searchString, languages) },
                ]
            })
        })
    },
    customQueries(input: StarSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.excludeTags === true ? { tagId: null } : {}),
        ])
    },
})

const forMapper = {
    [StarFor.Comment]: 'comment',
    [StarFor.Organization]: 'organization',
    [StarFor.Project]: 'project',
    [StarFor.Routine]: 'routine',
    [StarFor.Standard]: 'standard',
    [StarFor.Tag]: 'tag',
    [StarFor.User]: 'user',
}

const starMutater = (prisma: PrismaType) => ({
    async star(userId: string, input: StarInput): Promise<boolean> {
        prisma.star.findMany({
            where: {
                tagId: null
            }
        })
        // Define prisma type for object being starred
        const prismaFor = (prisma[forMapper[input.starFor] as keyof PrismaType] as any);
        // Check if object being starred exists
        const starringFor: null | { id: string, stars: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, stars: true } });
        if (!starringFor)
            throw new CustomError(CODE.ErrorUnknown, 'Could not find object being starred', { code: genErrorCode('0110') });
        // Check if star already exists on object by this user TODO fix for tags
        const star = await prisma.star.findFirst({
            where: {
                byId: userId,
                [`${forMapper[input.starFor]}Id`]: input.forId
            }
        })
        // If star already existed and we want to star, 
        // or if star did not exist and we don't want to star, skip
        if ((star && input.isStar) || (!star && !input.isStar)) return true;
        // If star did not exist and we want to star, create
        if (!star && input.isStar) {
            // Create
            await prisma.star.create({
                data: {
                    byId: userId,
                    [`${forMapper[input.starFor]}Id`]: input.forId
                }
            })
            // Increment star count on object
            await prismaFor.update({
                where: { id: input.forId },
                data: { stars: starringFor.stars + 1 }
            })
            // Log star event
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Star,
                object1Type: input.starFor,
                object1Id: input.forId,
            }).catch(error => logger.log(LogLevel.error, 'Failed creating "Star" log', { code: genErrorCode('0201'), error }));
        }
        // If star did exist and we don't want to star, delete
        else if (star && !input.isStar) {
            // Delete star
            await prisma.star.delete({ where: { id: star.id } })
            // Decrement star count on object
            await prismaFor.update({
                where: { id: input.forId },
                data: { stars: starringFor.stars - 1 }
            })
            // Log unstar event
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.RemoveStar,
                object1Type: input.starFor,
                object1Id: input.forId,
            }).catch(error => logger.log(LogLevel.error, 'Failed creating "Remove Star" log', { code: genErrorCode('0202'), error }));
        }
        return true;
    }
})

const starQuerier = (prisma: PrismaType) => ({
    async getIsStarreds(
        userId: string | null,
        ids: string[],
        starFor: keyof typeof StarFor
    ): Promise<boolean[]> {
        // Create result array that is the same length as ids
        const result = new Array(ids.length).fill(false);
        // If userId not passed, return result
        if (!userId) return result;
        // Filter out nulls and undefineds from ids
        const idsFiltered = onlyValidIds(ids);
        const fieldName = `${starFor.toLowerCase()}Id`;
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
        // Replace the nulls in the result array with true or false
        for (let i = 0; i < ids.length; i++) {
            // check if this id is in isStarredArray
            if (ids[i] !== null && ids[i] !== undefined &&
                isStarredArray.find((star: any) => star[fieldName] === ids[i])) {
                result[i] = true;
            }
        }
        return result;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const StarModel = ({
    prismaObject: (prisma: PrismaType) => prisma.star,
    format: starFormatter(),
    mutate: starMutater,
    query: starQuerier,
    search: starSearcher(),
    type: 'Star' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================