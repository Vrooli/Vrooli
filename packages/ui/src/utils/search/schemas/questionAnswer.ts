import { QuestionAnswerSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { questionAnswerFindMany } from "../../../api/generated/endpoints/questionAnswer_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionAnswerSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuestionAnswer"),
    containers: [], //TODO
    fields: [], //TODO
})

export const questionAnswerSearchParams = () => toParams(questionAnswerSearchSchema(), questionAnswerFindMany, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc)