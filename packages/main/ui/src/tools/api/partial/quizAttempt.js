import { rel } from "../utils";
export const quizAttemptYou = {
    __typename: "QuizAttemptYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};
export const quizAttempt = {
    __typename: "QuizAttempt",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        pointsEarned: true,
        status: true,
        contextSwitches: true,
        timeTaken: true,
        responsesCount: true,
        quiz: async () => rel((await import("./quiz")).quiz, "nav"),
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(quizAttemptYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./quizQuestionResponse")).quizQuestionResponse, "full"),
    },
    list: {},
};
//# sourceMappingURL=quizAttempt.js.map