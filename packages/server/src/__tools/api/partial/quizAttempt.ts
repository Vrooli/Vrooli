import { QuizAttempt, QuizAttemptYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const quizAttemptYou: ApiPartial<QuizAttemptYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const quizAttempt: ApiPartial<QuizAttempt> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        pointsEarned: true,
        status: true,
        contextSwitches: true,
        timeTaken: true,
        responsesCount: true,
        quiz: async () => rel((await import("./quiz.js")).quiz, "nav"),
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(quizAttemptYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./quizQuestionResponse.js")).quizQuestionResponse, "full"),
    },
};
