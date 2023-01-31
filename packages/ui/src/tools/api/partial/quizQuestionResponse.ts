import { QuizQuestionResponse, QuizQuestionResponseTranslation, QuizQuestionResponseYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const quizQuestionResponseTranslationPartial: GqlPartial<QuizQuestionResponseTranslation> = {
    __typename: 'QuizQuestionResponseTranslation',
    common: {
        id: true,
        language: true,
        response: true,
    },
    full: {},
    list: {},
}

export const quizQuestionResponseYouPartial: GqlPartial<QuizQuestionResponseYou> = {
    __typename: 'QuizQuestionResponseYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const quizQuestionResponsePartial: GqlPartial<QuizQuestionResponse> = {
    __typename: 'QuizQuestionResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: async () => relPartial((await import('./quizAttempt')).quizAttemptPartial, 'nav', { omit: 'responses' }),
        quizQuestion: async () => relPartial((await import('./quizQuestion')).quizQuestionPartial, 'nav', { omit: 'responses' }),
        translations: () => relPartial(quizQuestionResponseTranslationPartial, 'full'),
        you: () => relPartial(quizQuestionResponseYouPartial, 'full'),
    },
    full: {},
    list: {},
}