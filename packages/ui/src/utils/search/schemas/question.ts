import { QuestionSortBy } from "@local/shared";
import { questionFindMany } from "api/generated/endpoints/question_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchQuestion'),
    containers: [], //TODO
    fields: [], //TODO
})

export const questionSearchParams = () => toParams(questionSearchSchema(), questionFindMany, QuestionSortBy, QuestionSortBy.ScoreDesc)