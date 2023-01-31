import { Quiz, QuizTranslation, QuizYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const quizTranslationPartial: GqlPartial<QuizTranslation> = {
    __typename: 'QuizTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const quizYouPartial: GqlPartial<QuizYou> = {
    __typename: 'QuizYou',
    common: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const quizPartial: GqlPartial<Quiz> = {
    __typename: 'Quiz',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
        isCompleted: true,
        score: true,
        stars: true,
        views: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: async () => relPartial((await import('./project')).projectPartial, 'nav'),
        routine: async () => relPartial((await import('./routine')).routinePartial, 'nav'),
        you: async () => relPartial((await import('./quiz')).quizYouPartial, 'full'),
    },
    full: {
        quizQuestions: async () => relPartial((await import('./quizQuestion')).quizQuestionPartial, 'full', { omit: 'quiz' }),
        stats: async () => relPartial((await import('./statsQuiz')).statsQuizPartial, 'full'),
        translations: () => relPartial(quizTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizTranslationPartial, 'list'),
    }
}