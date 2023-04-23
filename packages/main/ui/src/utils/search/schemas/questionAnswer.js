import { QuestionAnswerSortBy } from "@local/consts";
import { questionAnswerFindMany } from "../../../api/generated/endpoints/questionAnswer_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const questionAnswerSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuestionAnswer"),
    containers: [],
    fields: [],
});
export const questionAnswerSearchParams = () => toParams(questionAnswerSearchSchema(), questionAnswerFindMany, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc);
//# sourceMappingURL=questionAnswer.js.map