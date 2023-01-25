import { QuizAttempt, QuizAttemptYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const quizAttemptYouPartial: GqlPartial<QuizAttemptYou> = {
    __typename: 'QuizAttemptYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
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
        quiz: () => relPartial(require('./quiz').quizPartial, 'nav'),
        user: () => relPartial(require('./user').userPartial, 'nav'),
        you: () => relPartial(quizAttemptYouPartial, 'full'),
    },
    full: {
        responses: () => relPartial(require('./quizQuestionResponse').quizQuestionResponsePartial, 'full'),
    },
}