import { Question, QuestionTranslation, QuestionYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { notePartial } from "./note";
import { organizationPartial } from "./organization";
import { projectPartial } from "./project";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";

export const questionTranslationPartial: GqlPartial<QuestionTranslation> = {
    __typename: 'QuestionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const questionYouPartial: GqlPartial<QuestionYou> = {
    __typename: 'QuestionYou',
    full: {
        isUpvoted: true,
    },
}

export const questionPartial: GqlPartial<Question> = {
    __typename: 'Question',
    common: {
        __define: {
            0: [apiPartial, 'nav'],
            1: [notePartial, 'nav'],
            2: [organizationPartial, 'nav'],
            3: [projectPartial, 'nav'],
            4: [routinePartial, 'nav'],
            5: [smartContractPartial, 'nav'],
            6: [standardPartial, 'nav'],
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