import { QuizQuestionResponse, QuizQuestionResponseTranslation, QuizQuestionResponseYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const quizQuestionResponseTranslation: GqlPartial<QuizQuestionResponseTranslation> = {
    __typename: 'QuizQuestionResponseTranslation',
    common: {
        id: true,
        language: true,
        response: true,
    },
    full: {},
    list: {},
}

export const quizQuestionResponseYou: GqlPartial<QuizQuestionResponseYou> = {
    __typename: 'QuizQuestionResponseYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const quizQuestionResponse: GqlPartial<QuizQuestionResponse> = {
    __typename: 'QuizQuestionResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: async () => rel((await import('./quizAttempt')).quizAttempt, 'nav', { omit: 'responses' }),
        quizQuestion: async () => rel((await import('./quizQuestion')).quizQuestion, 'nav', { omit: 'responses' }),
        translations: () => rel(quizQuestionResponseTranslation, 'full'),
        you: () => rel(quizQuestionResponseYou, 'full'),
    },
    full: {},
    list: {},
}