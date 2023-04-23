import { QuizQuestionResponseSortBy } from "@local/consts";
import { quizQuestionResponseFindMany } from "../../../api/generated/endpoints/quizQuestionResponse_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const quizQuestionResponseSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuizQuestionResponse"),
    containers: [],
    fields: [],
});
export const quizQuestionResponseSearchParams = () => toParams(quizQuestionResponseSearchSchema(), quizQuestionResponseFindMany, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc);
//# sourceMappingURL=quizQuestionResponse.js.map