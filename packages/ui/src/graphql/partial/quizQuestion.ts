import { QuizQuestion, QuizQuestionTranslation, QuizQuestionYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { quizPartial } from "./quiz";
import { quizQuestionResponsePartial } from "./quizQuestionResponse";
import { standardVersionPartial } from "./standardVersion";

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
        quiz: () => relPartial(quizPartial, 'nav', { omit: 'quizQuestions' }),
        standardVersion: () => relPartial(standardVersionPartial, 'nav'),
        you: () => relPartial(quizQuestionYouPartial, 'full'),
    },
    full: {
        responses: () => relPartial(quizQuestionResponsePartial, 'full', { omit: 'quizQuestion' }),
        translations: () => relPartial(quizQuestionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizQuestionTranslationPartial, 'list'),
    }
}