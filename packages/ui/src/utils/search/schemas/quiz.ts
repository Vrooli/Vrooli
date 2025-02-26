import { endpointsQuiz, FormSchema, QuizSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function quizSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchQuiz"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function quizSearchParams() {
    return toParams(quizSearchSchema(), endpointsQuiz, QuizSortBy, QuizSortBy.BookmarksDesc);
}
