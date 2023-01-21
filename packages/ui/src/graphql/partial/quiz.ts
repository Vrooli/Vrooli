import { Quiz, QuizTranslation, QuizYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { projectPartial } from "./project";
import { quizQuestionPartial } from "./quizQuestion";
import { routinePartial } from "./routine";
import { statsQuizPartial } from "./statsQuiz";
import { userPartial } from "./user";

export const quizTranslationPartial: GqlPartial<QuizTranslation> = {
    __typename: 'QuizTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const quizYouPartial: GqlPartial<QuizYou> = {
    __typename: 'QuizYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    },
}

export const quizPartial: GqlPartial<Quiz> = {
    __typename: 'Quiz',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: () => relPartial(userPartial, 'nav'),
        isCompleted: true,
        score: true,
        stars: true,
        views: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: () => relPartial(projectPartial, 'nav'),
        routine: () => relPartial(routinePartial, 'nav'),
        you: () => relPartial(quizYouPartial, 'full'),
    },
    full: {
        quizQuestions: () => relPartial(quizQuestionPartial, 'full', { omit: 'quiz' }),
        stats: () => relPartial(statsQuizPartial, 'full'),
        translations: () => relPartial(quizTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizTranslationPartial, 'list'),
    }
}