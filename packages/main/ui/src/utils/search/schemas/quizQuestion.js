import { QuizQuestionSortBy } from "@local/consts";
import { quizQuestionFindMany } from "../../../api/generated/endpoints/quizQuestion_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const quizQuestionSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuizQuestion"),
    containers: [],
    fields: [],
});
export const quizQuestionSearchParams = () => toParams(quizQuestionSearchSchema(), quizQuestionFindMany, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc);
//# sourceMappingURL=quizQuestion.js.map