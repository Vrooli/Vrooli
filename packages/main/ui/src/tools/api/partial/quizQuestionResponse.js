import { rel } from "../utils";
export const quizQuestionResponseYou = {
    __typename: "QuizQuestionResponseYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};
export const quizQuestionResponse = {
    __typename: "QuizQuestionResponse",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: async () => rel((await import("./quizAttempt")).quizAttempt, "nav", { omit: "responses" }),
        quizQuestion: async () => rel((await import("./quizQuestion")).quizQuestion, "nav", { omit: "responses" }),
        you: () => rel(quizQuestionResponseYou, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=quizQuestionResponse.js.map