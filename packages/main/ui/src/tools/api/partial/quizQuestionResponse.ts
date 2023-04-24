import { QuizQuestionResponse, QuizQuestionResponseYou } from ":local/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const quizQuestionResponseYou: GqlPartial<QuizQuestionResponseYou> = {
    __typename: "QuizQuestionResponseYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const quizQuestionResponse: GqlPartial<QuizQuestionResponse> = {
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
