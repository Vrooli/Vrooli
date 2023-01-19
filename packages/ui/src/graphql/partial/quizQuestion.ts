import { QuizQuestionTranslation, QuizQuestionYou } from "@shared/consts";
import { GqlPartial } from "types";

export const quizQuestionTranslationPartial: GqlPartial<QuizQuestionTranslation> = {
    __typename: 'QuizQuestionTranslation',
    full: {
        id: true,
        language: true,
        helpText: true,
        questionText: true,
    },
}

export const quizQuestionYouPartial: GqlPartial<QuizQuestionYou> = {
    __typename: 'QuizQuestionYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const quizQuestionFields = ['QuizQuestion', `{
    id
}`] as const;