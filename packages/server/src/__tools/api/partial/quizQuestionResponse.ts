import { QuizQuestionResponse, QuizQuestionResponseYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const quizQuestionResponseYou: ApiPartial<QuizQuestionResponseYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const quizQuestionResponse: ApiPartial<QuizQuestionResponse> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        response: true,
        quizAttempt: async () => rel((await import("./quizAttempt")).quizAttempt, "nav", { omit: "responses" }),
        quizQuestion: async () => rel((await import("./quizQuestion")).quizQuestion, "nav", { omit: "responses" }),
        you: () => rel(quizQuestionResponseYou, "full"),
    },
};
