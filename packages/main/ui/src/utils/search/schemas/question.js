import { QuestionSortBy } from "@local/consts";
import { questionFindMany } from "../../../api/generated/endpoints/question_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const questionSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuestion"),
    containers: [],
    fields: [],
});
export const questionSearchParams = () => toParams(questionSearchSchema(), questionFindMany, QuestionSortBy, QuestionSortBy.ScoreDesc);
//# sourceMappingURL=question.js.map