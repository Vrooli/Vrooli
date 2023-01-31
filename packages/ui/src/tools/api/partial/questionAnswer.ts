import { QuestionAnswer, QuestionAnswerTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const questionAnswerTranslationPartial: GqlPartial<QuestionAnswerTranslation> = {
    __typename: 'QuestionAnswerTranslation',
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
}

export const questionAnswerPartial: GqlPartial<QuestionAnswer> = {
    __typename: 'QuestionAnswer',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
        score: true,
        stars: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: async () => relPartial((await import('./comment')).comment, 'full', { omit: 'commentedOn' }),
        question: async () => relPartial((await import('./question')).questionPartial, 'nav', { omit: 'answers' }),
        translations: () => relPartial(questionAnswerTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(questionAnswerTranslationPartial, 'list'),
    }
}