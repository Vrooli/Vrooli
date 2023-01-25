import { QuizQuestionResponse, QuizQuestionResponseTranslation, QuizQuestionResponseYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const quizQuestionResponseTranslationPartial: GqlPartial<QuizQuestionResponseTranslation> = {
    __typename: 'QuizQuestionResponseTranslation',
    full: {
        id: true,
        language: true,
        response: true,
    },
}

export const quizQuestionResponseYouPartial: GqlPartial<QuizQuestionResponseYou> = {
    __typename: 'QuizQuestionResponseYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const quizQuestionResponsePartial: GqlPartial<QuizQuestionResponse> = {
    __typename: 'QuizQuestionResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: () => relPartial(require('./quizAttempt').quizAttemptPartial, 'nav', { omit: 'responses' }),
        quizQuestion: () => relPartial(require('./quizQuestion').quizQuestionPartial, 'nav', { omit: 'responses' }),
        translations: () => relPartial(quizQuestionResponseTranslationPartial, 'full'),
        you: () => relPartial(quizQuestionResponseYouPartial, 'full'),
    },
}