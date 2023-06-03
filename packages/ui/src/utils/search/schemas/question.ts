import { QuestionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuestion"),
    containers: [], //TODO
    fields: [], //TODO
});

export const questionSearchParams = () => toParams(questionSearchSchema(), "/questions", QuestionSortBy, QuestionSortBy.ScoreDesc);
