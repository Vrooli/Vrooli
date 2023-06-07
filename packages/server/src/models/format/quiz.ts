import { QuizModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Quiz" as const;
export const QuizFormat: Formatter<QuizModelLogic> = {
    gqlRelMap: {
        __typename,
        attempts: "QuizAttempt",
        createdBy: "User",
        project: "Project",
        quizQuestions: "QuizQuestion",
        routine: "Routine",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename,
        attempts: "QuizAttempt",
        createdBy: "User",
        project: "Project",
        quizQuestions: "QuizQuestion",
        routine: "Routine",
        bookmarkedBy: "User",
    },
    joinMap: { bookmarkedBy: "user" },
    countFields: {
        attemptsCount: true,
        quizQuestionsCount: true,
    },
};