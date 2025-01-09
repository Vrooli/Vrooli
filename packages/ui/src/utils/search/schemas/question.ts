import { endpointsQuestion, FormSchema, QuestionSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function questionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchQuestion"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function questionSearchParams() {
    return toParams(questionSearchSchema(), endpointsQuestion, QuestionSortBy, QuestionSortBy.ScoreDesc);
}
