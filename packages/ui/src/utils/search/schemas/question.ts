import { endpointGetQuestion, endpointGetQuestions, QuestionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuestion"),
    containers: [], //TODO
    elements: [], //TODO
});

export const questionSearchParams = () => toParams(questionSearchSchema(), endpointGetQuestions, endpointGetQuestion, QuestionSortBy, QuestionSortBy.ScoreDesc);
