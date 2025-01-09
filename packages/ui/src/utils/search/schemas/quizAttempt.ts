import { endpointsQuizAttempt, FormSchema, QuizAttemptSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
