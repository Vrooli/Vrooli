import { QuizAttempt, QuizAttemptYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const quizAttemptYouPartial: GqlPartial<QuizAttemptYou> = {
    __typename: 'QuizAttemptYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const quizAttemptPartial: GqlPartial<QuizAttempt> = {
    __typename: 'QuizAttempt',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        pointsEarned: true,
        status: true,
        contextSwitches: true,
        timeTaken: true,
        responsesCount: true,
        quiz: async () => relPartial((await import('./quiz')).quizPartial, 'nav'),
        user: async () => relPartial((await import('./user')).userPartial, 'nav'),
        you: () => relPartial(quizAttemptYouPartial, 'full'),
    },
    full: {
        responses: async () => relPartial((await import('./quizQuestionResponse')).quizQuestionResponsePartial, 'full'),
    },
    list: {},
}