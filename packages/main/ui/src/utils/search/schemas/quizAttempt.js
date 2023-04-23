import { QuizAttemptSortBy } from "@local/consts";
import { quizAttemptFindMany } from "../../../api/generated/endpoints/quizAttempt_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const quizAttemptSearchSchema = () => ({
    formLayout: searchFormLayout("SearchQuizAttempt"),
    containers: [],
    fields: [],
});
export const quizAttemptSearchParams = () => toParams(quizAttemptSearchSchema(), quizAttemptFindMany, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc);
//# sourceMappingURL=quizAttempt.js.map