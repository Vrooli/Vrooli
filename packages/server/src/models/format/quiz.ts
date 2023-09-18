import { QuizModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuizFormat: Formatter<QuizModelLogic> = {
    gqlRelMap: {
        __typename: "Quiz",
        attempts: "QuizAttempt",
        createdBy: "User",
        project: "Project",
        quizQuestions: "QuizQuestion",
        routine: "Routine",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Quiz",
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
