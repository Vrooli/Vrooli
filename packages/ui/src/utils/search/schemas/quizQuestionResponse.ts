import { endpointsQuizQuestionResponse, FormSchema, QuizQuestionResponseSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function quizQuestionResponseSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchQuizQuestionResponse"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function quizQuestionResponseSearchParams() {
    return toParams(quizQuestionResponseSearchSchema(), endpointsQuizQuestionResponse, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc);
}
