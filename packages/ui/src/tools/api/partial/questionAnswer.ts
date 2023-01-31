import { QuestionAnswer, QuestionAnswerTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const questionAnswerTranslation: GqlPartial<QuestionAnswerTranslation> = {
    __typename: 'QuestionAnswerTranslation',
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
}

export const questionAnswer: GqlPartial<QuestionAnswer> = {
    __typename: 'QuestionAnswer',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import('./user')).user, 'nav'),
        score: true,
        stars: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: async () => rel((await import('./comment')).comment, 'full', { omit: 'commentedOn' }),
        question: async () => rel((await import('./question')).question, 'nav', { omit: 'answers' }),
        translations: () => rel(questionAnswerTranslation, 'full'),
    },
    list: {
        translations: () => rel(questionAnswerTranslation, 'list'),
    }
}