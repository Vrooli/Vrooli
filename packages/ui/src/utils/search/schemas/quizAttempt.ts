import { endpointsQuizAttempt, FormSchema, QuizAttemptSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function quizAttemptSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchQuizAttempt"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function quizAttemptSearchParams() {
    return toParams(quizAttemptSearchSchema(), endpointsQuizAttempt, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc);
}
