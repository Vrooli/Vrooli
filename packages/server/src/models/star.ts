import { StarFor, StarSortBy } from "@shared/consts";
import { isObject } from '@shared/utils';
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { CommentModel } from "./comment";
import { CustomError, Trigger } from "../events";
import { resolveStarTo } from "../schema/resolvers";
import { Star, StarSearchInput, StarInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { readManyHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType, Searcher } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from ".";
import { PartialGraphQLInfo } from "../builders/types";
import { combineQueries, onlyValidIds, padSelect } from "../builders";
import { getDelegator } from "../getters";

const formatter = (): Formatter<Star, 'to'> => ({
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
        toGraphQL: ({ languages, objects, partial, prisma, userData }) => [
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
                    for (const objectType of Object.keys(toIdsByType)) {
                        const validTypes: GraphQLModelType[] = [
                            'Comment',
                            'Organization',
                            'Project',
                            'Routine',
                            'Standard',
                            'Tag',
                            'User',
                        ];
                        if (!validTypes.includes(objectType as GraphQLModelType)) {
                            throw new CustomError('0185', 'InternalError', languages, { objectType });
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
                return objects;
            }],
        ],
    },
})

const searcher = (): Searcher<
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

const forMapper: { [key in StarFor]: keyof Prisma.starUpsertArgs['create'] } = {
    Comment: 'comment',
    Organization: 'organization',
    Project: 'project',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user',
}

const star = async (prisma: PrismaType, userData: SessionUser, input: StarInput): Promise<boolean> => {
    prisma.star.findMany({
        where: {
            tagId: null
        }
    })
    // Get prisma delegate for type of object being starred
    const prismaDelegate = getDelegator(input.starFor, prisma, userData.languages, 'star');
    // Check if object being starred exists
    const starringFor: null | { id: string, stars: number } = await prismaDelegate.findUnique({ where: { id: input.forId }, select: { id: true, stars: true } }) as any;
    if (!starringFor)
        throw new CustomError('0110', 'ErrorUnknown', userData.languages, { starFor: input.starFor, forId: input.forId });
    // Check if star already exists on object by this user TODO fix for tags
    const star = await prisma.star.findFirst({
        where: {
            byId: userData.id,
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
                byId: userData.id,
                [`${forMapper[input.starFor]}Id`]: input.forId
            }
        })
        // Increment star count on object
        await prismaDelegate.update({
            where: { id: input.forId },
            data: { stars: starringFor.stars + 1 }
        })
        // Handle trigger
        await Trigger(prisma, userData.languages).objectStar(true, input.starFor, input.forId, userData.id);
    }
    // If star did exist and we don't want to star, delete
    else if (star && !input.isStar) {
        // Delete star
        await prisma.star.delete({ where: { id: star.id } })
        // Decrement star count on object
        await prismaDelegate.update({
            where: { id: input.forId },
            data: { stars: starringFor.stars - 1 }
        })
        // Handle trigger
        await Trigger(prisma, userData.languages).objectStar(false, input.starFor, input.forId, userData.id);
    }
    return true;
}

const querier = () => ({
    async getIsStarreds(
        prisma: PrismaType,
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

const displayer = (): Displayer<
    Prisma.starSelect,
    Prisma.starGetPayload<{ select: { [K in keyof Required<Prisma.starSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        // api: padSelect(ApiModel.display.select),
        comment: padSelect(CommentModel.display.select),
        // issue: padSelect(IssueModel.display.select),
        organization: padSelect(OrganizationModel.display.select),
        // post: padSelect(PostModel.display.select),
        project: padSelect(ProjectModel.display.select),
        // question: padSelect(QuestionModel.display.select),
        // questionAnswer: padSelect(QuestionAnswerModel.display.select),
        // quiz: padSelect(QuizModel.display.select),
        routine: padSelect(RoutineModel.display.select),
        // smartContract: padSelect(SmartContractModel.display.select),
        standard: padSelect(StandardModel.display.select),
        tag: padSelect(TagModel.display.select),
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => {
        // if (select.api) return ApiModel.display.label(select.api as any, languages);
        if (select.comment) return CommentModel.display.label(select.comment as any, languages);
        // if (select.issue) return IssueModel.display.label(select.issue as any, languages);
        if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
        // if (select.post) return PostModel.display.label(select.post as any, languages);
        if (select.project) return ProjectModel.display.label(select.project as any, languages);
        // if (select.question) return QuestionModel.display.label(select.question as any, languages);
        // if (select.questionAnswer) return QuestionAnswerModel.display.label(select.questionAnswer as any, languages);
        // if (select.quiz) return QuizModel.display.label(select.quiz as any, languages);
        if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
        // if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
        if (select.standard) return StandardModel.display.label(select.standard as any, languages);
        if (select.tag) return TagModel.display.label(select.tag as any, languages);
        if (select.user) return UserModel.display.label(select.user as any, languages);
        return '';
    }
})

export const StarModel = ({
    delegate: (prisma: PrismaType) => prisma.star,
    display: displayer(),
    format: formatter(),
    query: querier(),
    search: searcher(),
    type: 'Star' as GraphQLModelType,
    star,
})