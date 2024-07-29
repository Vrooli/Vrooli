import { endpointGetQuestionAnswer, endpointGetQuestionAnswers, FormSchema, QuestionAnswerSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionAnswerSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuestionAnswer"),
    containers: [], //TODO
    elements: [], //TODO
});

export const questionAnswerSearchParams = () => toParams(questionAnswerSearchSchema(), endpointGetQuestionAnswers, endpointGetQuestionAnswer, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc);
