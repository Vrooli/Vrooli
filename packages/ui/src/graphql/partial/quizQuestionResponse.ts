import { QuizQuestionResponseTranslation, QuizQuestionResponseYou } from "@shared/consts";
import { GqlPartial } from "types";

export const quizQuestionResponseTranslationPartial: GqlPartial<QuizQuestionResponseTranslation> = {
    __typename: 'QuizQuestionResponseTranslation',
    full: {
        id: true,
        language: true,
        response: true,
    },
}

export const quizQuestionRepsonseYouPartial: GqlPartial<QuizQuestionResponseYou> = {
    __typename: 'QuizQuestionResponseYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const quizQuestionResponseFields = ['QuizQuestionResponse', `{
    id
}`] as const;