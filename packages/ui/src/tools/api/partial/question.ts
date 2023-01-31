import { Question, QuestionTranslation, QuestionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const questionTranslationPartial: GqlPartial<QuestionTranslation> = {
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

export const questionYouPartial: GqlPartial<QuestionYou> = {
    __typename: 'QuestionYou',
    common: {
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const questionPartial: GqlPartial<Question> = {
    __typename: 'Question',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'nav'),
            1: async () => relPartial((await import('./note')).notePartial, 'nav'),
            2: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
            3: async () => relPartial((await import('./project')).projectPartial, 'nav'),
            4: async () => relPartial((await import('./routine')).routinePartial, 'nav'),
            5: async () => relPartial((await import('./smartContract')).smartContractPartial, 'nav'),
            6: async () => relPartial((await import('./standard')).standardPartial, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
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
        you: () => relPartial(questionYouPartial, 'full'),
    },
    full: {
        answers: async () => relPartial((await import('./questionAnswer')).questionAnswerPartial, 'full', { omit: 'question' }),
        translations: () => relPartial(questionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(questionTranslationPartial, 'list'),
    }
}