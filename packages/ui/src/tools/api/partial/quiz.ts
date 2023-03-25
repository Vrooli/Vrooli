import { Quiz, QuizTranslation, QuizYou } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const quizTranslation: GqlPartial<QuizTranslation> = {
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

export const quizYou: GqlPartial<QuizYou> = {
    __typename: 'QuizYou',
    common: {
        canDelete: true,
        canBookmark: true,
        canUpdate: true,
        canRead: true,
        canVote: true,
        hasCompleted: true,
        isBookmarked: true,
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const quiz: GqlPartial<Quiz> = {
    __typename: 'Quiz',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import('./user')).user, 'nav'),
        score: true,
        bookmarks: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: async () => rel((await import('./project')).project, 'nav'),
        routine: async () => rel((await import('./routine')).routine, 'nav'),
        you: async () => rel((await import('./quiz')).quizYou, 'full'),
    },
    full: {
        quizQuestions: async () => rel((await import('./quizQuestion')).quizQuestion, 'full', { omit: 'quiz' }),
        stats: async () => rel((await import('./statsQuiz')).statsQuiz, 'full'),
        translations: () => rel(quizTranslation, 'full'),
    },
    list: {
        translations: () => rel(quizTranslation, 'list'),
    }
}