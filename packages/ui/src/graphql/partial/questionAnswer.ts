import { QuestionAnswer, QuestionAnswerTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { commentPartial } from "./comment";
import { questionPartial } from "./question";
import { userPartial } from "./user";

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
        createdBy: () => relPartial(userPartial, 'nav'),
        score: true,
        stars: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: () => relPartial(commentPartial, 'full', { omit: 'commentedOn' }),
        question: () => relPartial(questionPartial, 'nav', { omit: 'answers' }),
        translations: () => relPartial(questionAnswerTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(questionAnswerTranslationPartial, 'list'),
    }
}