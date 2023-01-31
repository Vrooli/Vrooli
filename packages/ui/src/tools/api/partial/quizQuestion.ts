import { QuizQuestion, QuizQuestionTranslation, QuizQuestionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const quizQuestionTranslationPartial: GqlPartial<QuizQuestionTranslation> = {
    __typename: 'QuizQuestionTranslation',
    common: {
        id: true,
        language: true,
        helpText: true,
        questionText: true,
    },
    full: {},
    list: {},
}

export const quizQuestionYouPartial: GqlPartial<QuizQuestionYou> = {
    __typename: 'QuizQuestionYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const quizQuestionPartial: GqlPartial<QuizQuestion> = {
    __typename: 'QuizQuestion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        order: true,
        points: true,
        responsesCount: true,
        quiz: async () => relPartial((await import('./quiz')).quizPartial, 'nav', { omit: 'quizQuestions' }),
        standardVersion: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'nav'),
        you: () => relPartial(quizQuestionYouPartial, 'full'),
    },
    full: {
        responses: async () => relPartial((await import('./quizQuestionResponse')).quizQuestionResponsePartial, 'full', { omit: 'quizQuestion' }),
        translations: () => relPartial(quizQuestionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizQuestionTranslationPartial, 'list'),
    }
}