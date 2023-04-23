import { QuizSortBy } from "@local/consts";
import { quizFindMany } from "../../../api/generated/endpoints/quiz_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const quizSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuiz"),
    containers: [],
    fields: [],
});
export const quizSearchParams = () => toParams(quizSearchSchema(), quizFindMany, QuizSortBy, QuizSortBy.BookmarksDesc);
//# sourceMappingURL=quiz.js.map