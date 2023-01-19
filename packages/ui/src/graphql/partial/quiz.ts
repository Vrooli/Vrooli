import { QuizTranslation, QuizYou } from "@shared/consts";
import { GqlPartial } from "types";

export const quizTranslationPartial: GqlPartial<QuizTranslation> = {
    __typename: 'QuizTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
        name: true,
    }),
}

export const quizYouPartial: GqlPartial<QuizYou> = {
    __typename: 'QuizYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canStar: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    }),
}

export const listQuizFields = ['Quiz', `{
    id
}`] as const;
export const quizFields = ['Quiz', `{
    id
}`] as const;