import { QuizQuestionResponse, QuizQuestionResponseYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const quizQuestionResponseYou: GqlPartial<QuizQuestionResponseYou> = {
    __typename: 'QuizQuestionResponseYou',
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
}

export const quizQuestionResponse: GqlPartial<QuizQuestionResponse> = {
    __typename: 'QuizQuestionResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: async () => rel((await import('./quizAttempt')).quizAttempt, 'nav', { omit: 'responses' }),
        quizQuestion: async () => rel((await import('./quizQuestion')).quizQuestion, 'nav', { omit: 'responses' }),
        you: () => rel(quizQuestionResponseYou, 'full'),
    },
    full: {},
    list: {},
}