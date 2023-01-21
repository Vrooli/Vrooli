import { QuizQuestionResponse, QuizQuestionResponseTranslation, QuizQuestionResponseYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { quizAttemptPartial } from "./quizAttempt";
import { quizQuestionPartial } from "./quizQuestion";

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
        quizAttempt: () => relPartial(quizAttemptPartial, 'nav', { omit: 'responses' }),
        quizQuestion: () => relPartial(quizQuestionPartial, 'nav', { omit: 'responses' }),
        you: () => relPartial(quizQuestionResponseYouPartial, 'full'),
    },
}