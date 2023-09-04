import { endpointGetQuestionAnswer, endpointGetQuestionAnswers, QuestionAnswerSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionAnswerSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuestionAnswer"),
    containers: [], //TODO
    fields: [], //TODO
});

export const questionAnswerSearchParams = () => toParams(questionAnswerSearchSchema(), endpointGetQuestionAnswers, endpointGetQuestionAnswer, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc);
