import { Question, QuestionTranslation, QuestionYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const questionTranslation: GqlPartial<QuestionTranslation> = {
    __typename: 'QuestionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const questionYou: GqlPartial<QuestionYou> = {
    __typename: 'QuestionYou',
    common: {
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const question: GqlPartial<Question> = {
    __typename: 'Question',
    common: {
        __define: {
            0: async () => rel((await import('./api')).api, 'nav'),
            1: async () => rel((await import('./note')).note, 'nav'),
            2: async () => rel((await import('./organization')).organization, 'nav'),
            3: async () => rel((await import('./project')).project, 'nav'),
            4: async () => rel((await import('./routine')).routine, 'nav'),
            5: async () => rel((await import('./smartContract')).smartContract, 'nav'),
            6: async () => rel((await import('./standard')).standard, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import('./user')).user, 'nav'),
        hasAcceptedAnswer: true,
        score: true,
        stars: true,
        answersCount: true,
        commentsCount: true,
        forObject: {
            __union: {
                Api: 0,
                Note: 1,
                Organization: 2,
                Project: 3,
                Routine: 4,
                SmartContract: 5,
                Standard: 6,
            }
        },
        you: () => rel(questionYou, 'full'),
    },
    full: {
        answers: async () => rel((await import('./questionAnswer')).questionAnswer, 'full', { omit: 'question' }),
        translations: () => rel(questionTranslation, 'full'),
    },
    list: {
        translations: () => rel(questionTranslation, 'list'),
    }
}