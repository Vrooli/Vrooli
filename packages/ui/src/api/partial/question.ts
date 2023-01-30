import { Question, QuestionTranslation, QuestionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
            0: () => relPartial(require('./api').apiPartial, 'nav'),
            1: () => relPartial(require('./note').notePartial, 'nav'),
            2: () => relPartial(require('./organization').organizationPartial, 'nav'),
            3: () => relPartial(require('./project').projectPartial, 'nav'),
            4: () => relPartial(require('./routine').routinePartial, 'nav'),
            5: () => relPartial(require('./smartContract').smartContractPartial, 'nav'),
            6: () => relPartial(require('./standard').standardPartial, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
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
        answers: () => relPartial(require('./questionAnswer').questionAnswerPartial, 'full', { omit: 'question' }),
        translations: () => relPartial(questionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(questionTranslationPartial, 'list'),
    }
}