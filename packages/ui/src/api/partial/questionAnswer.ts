import { QuestionAnswer, QuestionAnswerTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const questionAnswerTranslationPartial: GqlPartial<QuestionAnswerTranslation> = {
    __typename: 'QuestionAnswerTranslation',
    full: {
        id: true,
        language: true,
        description: true,
    },
}

export const questionAnswerPartial: GqlPartial<QuestionAnswer> = {
    __typename: 'QuestionAnswer',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
        score: true,
        stars: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: () => relPartial(require('./comment').commentPartial, 'full', { omit: 'commentedOn' }),
        question: () => relPartial(require('./question').questionPartial, 'nav', { omit: 'answers' }),
        translations: () => relPartial(questionAnswerTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(questionAnswerTranslationPartial, 'list'),
    }
}