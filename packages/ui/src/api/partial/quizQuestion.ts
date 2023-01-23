import { QuizQuestion, QuizQuestionTranslation, QuizQuestionYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const quizQuestionTranslationPartial: GqlPartial<QuizQuestionTranslation> = {
    __typename: 'QuizQuestionTranslation',
    full: {
        id: true,
        language: true,
        helpText: true,
        questionText: true,
    },
}

export const quizQuestionYouPartial: GqlPartial<QuizQuestionYou> = {
    __typename: 'QuizQuestionYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
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
        quiz: () => relPartial(require('./quiz').quizPartial, 'nav', { omit: 'quizQuestions' }),
        standardVersion: () => relPartial(require('./standardVersion').standardVersionPartial, 'nav'),
        you: () => relPartial(quizQuestionYouPartial, 'full'),
    },
    full: {
        responses: () => relPartial(require('./quizQuestionResponse').quizQuestionResponsePartial, 'full', { omit: 'quizQuestion' }),
        translations: () => relPartial(quizQuestionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizQuestionTranslationPartial, 'list'),
    }
}