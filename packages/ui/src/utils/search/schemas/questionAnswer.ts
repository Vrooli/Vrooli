import { endpointsQuestionAnswer, FormSchema, QuestionAnswerSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function questionAnswerSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchQuestionAnswer"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function questionAnswerSearchParams() {
    return toParams(questionAnswerSearchSchema(), endpointsQuestionAnswer, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc);
}
