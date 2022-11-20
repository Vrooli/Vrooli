import { StarFor, StarSortBy } from "@shared/consts";
import { isObject } from '@shared/utils';
import { combineQueries, ObjectMap, onlyValidIds } from "./builder";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { CommentModel } from "./comment";
import { CustomError, Trigger } from "../events";
import { resolveStarTo } from "../schema/resolvers";
import { Star, StarSearchInput, StarInput } from "../schema/types";
import { PrismaType } from "../types";
import { readManyHelper } from "./actions";
import { FormatConverter, GraphQLModelType, ModelLogic, Mutater, PartialGraphQLInfo, Searcher } from "./types";
import { Prisma } from "@prisma/client";

export const starFormatter = (): FormatConverter<Star, 'to'> => ({
    relationshipMap: {
        __typename: 'Star',
        from: 'User',
        to: {
            Comment: 'Comment',
            Organization: 'Organization',
            Project: 'Project',
            Routine: 'Routine',
            Standard: 'Standard',
            Tag: 'Tag',
            User: 'User',
        }
    },
    supplemental: {
        graphqlFields: ['to'],
        toGraphQL: ({ objects, partial, prisma, userData }) => [
            ['to', async () => {
                if (!userData) return new Array(objects.length).fill([]);
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
                        const validTypes: GraphQLModelType[] = [
                            'Comment',
                            'Organization',
                            'Project',
                            'Routine',
                            'Standard',
                            'Tag',
                            'User',
                        ];
                        if (!validTypes.includes(type as GraphQLModelType)) {
                            throw new CustomError('InternalError', `View applied to unsupported type: ${type}`, { trace: '0185' });
                        }
                        const model: ModelLogic<any, any, any, any> = ObjectMap[type] as ModelLogic<any, any, any, any>;
                        const paginated = await readManyHelper({
                            info: partial.to[type] as PartialGraphQLInfo,
                            input: { ids: toIdsByType[type] },
                            model,
                            prisma,
                            req: { users: [userData] }
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
            }],
        ],
    },
})

export const starSearcher = (): Searcher<
    StarSearchInput,
    StarSortBy,
    Prisma.starOrderByWithRelationInput,
    Prisma.starWhereInput
> => ({
    defaultSort: StarSortBy.DateUpdatedDesc,
    sortMap: {
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages, searchString }) => ({
        OR: [
            { organization: OrganizationModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { project: ProjectModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { routine: RoutineModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { standard: StandardModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { tag: TagModel.search.searchStringQuery({ insensitive, languages, searchString }) },
            { comment: CommentModel.search.searchStringQuery({ insensitive, languages, searchString }) },
        ]
    }),
    customQueries(input) {
        return combineQueries([
            (input.excludeTags === true ? { tagId: null } : {}),
        ])
    },
})

const forMapper: { [key in StarFor]: string } = {
    Comment: 'comment',
    Organization: 'organization',
    Project: 'project',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user',
}

const starMutater = (prisma: PrismaType): Mutater<Star> => ({
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
            throw new CustomError('ErrorUnknown', 'Could not find object being starred', { trace: '0110' });
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
            // Handle trigger
            await Trigger(prisma).objectStar(true, input.starFor, input.forId, userId);
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
            // Handle trigger
            await Trigger(prisma).objectStar(false, input.starFor, input.forId, userId);
        }
        return true;
    }
})

const starQuerier = (prisma: PrismaType) => ({
    async getIsStarreds(
        userId: string | null | undefined,
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

export const StarModel = ({
    prismaObject: (prisma: PrismaType) => prisma.star,
    format: starFormatter(),
    mutate: starMutater,
    query: starQuerier,
    search: starSearcher(),
    type: 'Star' as GraphQLModelType,
})