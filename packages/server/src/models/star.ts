import { StarFor, StarSortBy } from "@shared/consts";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { CommentModel } from "./comment";
import { CustomError, Trigger } from "../events";
import { Star, StarSearchInput, StarInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic, Searcher } from "./types";
import { Prisma } from "@prisma/client";
import { ApiModel, IssueModel, PostModel, QuestionAnswerModel, QuestionModel, QuizModel, SmartContractModel, UserModel } from ".";
import { SelectWrap } from "../builders/types";
import { onlyValidIds, padSelect } from "../builders";
import { getLogic } from "../getters";
import { NoteModel } from "./note";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Star,
    GqlSearch: StarSearchInput,
    GqlSort: StarSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.starUpsertArgs['create'],
    PrismaUpdate: Prisma.starUpsertArgs['update'],
    PrismaModel: Prisma.starGetPayload<SelectWrap<Prisma.starSelect>>,
    PrismaSelect: Prisma.starSelect,
    PrismaWhere: Prisma.starWhereInput,
}

const __typename = 'Star' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        by: 'User',
        to: {
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
            note: 'Note',
            organization: 'Organization',
            post: 'Post',
            project: 'Project',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            quiz: 'Quiz',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            tag: 'Tag',
            user: 'User',
        }
    },
    prismaRelMap: {
        __typename,
        by: 'User',
        api: 'Api',
        comment: 'Comment',
        issue: 'Issue',
        note: 'Note',
        organization: 'Organization',
        post: 'Post',
        project: 'Project',
        question: 'Question',
        questionAnswer: 'QuestionAnswer',
        quiz: 'Quiz',
        routine: 'Routine',
        smartContract: 'SmartContract',
        standard: 'Standard',
        tag: 'Tag',
        user: 'User',
    },
    countFields: {},
})

const searcher = (): Searcher<Model> => ({
    defaultSort: StarSortBy.DateUpdatedDesc,
    sortBy: StarSortBy,
    searchFields: {
        excludeLinkedToTag: true,
    },
    searchStringQuery: () => ({
        OR: [
            { api: ApiModel.search!.searchStringQuery() },
            { comment: CommentModel.search!.searchStringQuery() },
            { issue: IssueModel.search!.searchStringQuery() },
            { note: NoteModel.search!.searchStringQuery() },
            { organization: OrganizationModel.search!.searchStringQuery() },
            { post: PostModel.search!.searchStringQuery() },
            { project: ProjectModel.search!.searchStringQuery() },
            { question: QuestionModel.search!.searchStringQuery() },
            { questionAnswer: QuestionAnswerModel.search!.searchStringQuery() },
            { quiz: QuizModel.search!.searchStringQuery() },
            { routine: RoutineModel.search!.searchStringQuery() },
            { smartContract: SmartContractModel.search!.searchStringQuery() },
            { standard: StandardModel.search!.searchStringQuery() },
            { tag: TagModel.search!.searchStringQuery() },
            { user: UserModel.search!.searchStringQuery() },
        ]
    }),
})

const forMapper: { [key in StarFor]: keyof Prisma.starUpsertArgs['create'] } = {
    Api: 'api',
    Comment: 'comment',
    Issue: 'issue',
    Note: 'note',
    Organization: 'organization',
    Post: 'post',
    Project: 'project',
    Question: 'question',
    QuestionAnswer: 'questionAnswer',
    Quiz: 'quiz',
    Routine: 'routine',
    SmartContract: 'smartContract',
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
    const { delegate } = getLogic(['delegate'], input.starFor, userData.languages, 'star');
    // Check if object being starred exists
    const starringFor: null | { id: string, stars: number } = await delegate(prisma).findUnique({ where: { id: input.forConnect }, select: { id: true, stars: true } }) as any;
    if (!starringFor)
        throw new CustomError('0110', 'ErrorUnknown', userData.languages, { starFor: input.starFor, forId: input.forConnect });
    // Check if star already exists on object by this user TODO fix for tags
    const star = await prisma.star.findFirst({
        where: {
            byId: userData.id,
            [`${forMapper[input.starFor]}Id`]: input.forConnect
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
                [`${forMapper[input.starFor]}Id`]: input.forConnect
            }
        })
        // Increment star count on object
        await delegate(prisma).update({
            where: { id: input.forConnect },
            data: { stars: starringFor.stars + 1 }
        })
        // Handle trigger
        await Trigger(prisma, userData.languages).objectStar(true, input.starFor, input.forConnect, userData.id);
    }
    // If star did exist and we don't want to star, delete
    else if (star && !input.isStar) {
        // Delete star
        await prisma.star.delete({ where: { id: star.id } })
        // Decrement star count on object
        await delegate(prisma).update({
            where: { id: input.forConnect },
            data: { stars: starringFor.stars - 1 }
        })
        // Handle trigger
        await Trigger(prisma, userData.languages).objectStar(false, input.starFor, input.forConnect, userData.id);
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

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        api: padSelect(ApiModel.display.select),
        comment: padSelect(CommentModel.display.select),
        issue: padSelect(IssueModel.display.select),
        note: padSelect(NoteModel.display.select),
        organization: padSelect(OrganizationModel.display.select),
        post: padSelect(PostModel.display.select),
        project: padSelect(ProjectModel.display.select),
        question: padSelect(QuestionModel.display.select),
        questionAnswer: padSelect(QuestionAnswerModel.display.select),
        quiz: padSelect(QuizModel.display.select),
        routine: padSelect(RoutineModel.display.select),
        smartContract: padSelect(SmartContractModel.display.select),
        standard: padSelect(StandardModel.display.select),
        tag: padSelect(TagModel.display.select),
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => {
        if (select.api) return ApiModel.display.label(select.api as any, languages);
        if (select.comment) return CommentModel.display.label(select.comment as any, languages);
        if (select.issue) return IssueModel.display.label(select.issue as any, languages);
        if (select.note) return NoteModel.display.label(select.note as any, languages);
        if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
        if (select.post) return PostModel.display.label(select.post as any, languages);
        if (select.project) return ProjectModel.display.label(select.project as any, languages);
        if (select.question) return QuestionModel.display.label(select.question as any, languages);
        if (select.questionAnswer) return QuestionAnswerModel.display.label(select.questionAnswer as any, languages);
        if (select.quiz) return QuizModel.display.label(select.quiz as any, languages);
        if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
        if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
        if (select.standard) return StandardModel.display.label(select.standard as any, languages);
        if (select.tag) return TagModel.display.label(select.tag as any, languages);
        if (select.user) return UserModel.display.label(select.user as any, languages);
        return '';
    }
})

export const StarModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.star,
    display: displayer(),
    format: formatter(),
    query: querier(),
    search: searcher(),
    star,
})