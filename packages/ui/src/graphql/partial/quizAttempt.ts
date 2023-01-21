import { QuizAttempt, QuizAttemptYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { quizPartial } from "./quiz";
import { quizQuestionResponsePartial } from "./quizQuestionResponse";
import { userPartial } from "./user";

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
        quiz: () => relPartial(quizPartial, 'nav'),
        user: () => relPartial(userPartial, 'nav'),
        you: () => relPartial(quizAttemptYouPartial, 'full'),
    },
    full: {
        responses: () => relPartial(quizQuestionResponsePartial, 'full'),
    },
}