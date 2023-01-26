import { Quiz, QuizTranslation, QuizYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
        isCompleted: true,
        score: true,
        stars: true,
        views: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: () => relPartial(require('./project').projectPartial, 'nav'),
        routine: () => relPartial(require('./routine').routinePartial, 'nav'),
        you: () => relPartial(require('./quiz').quizYouPartial, 'full'),
    },
    full: {
        quizQuestions: () => relPartial(require('./quizQuestion').quizQuestionPartial, 'full', { omit: 'quiz' }),
        stats: () => relPartial(require('./statsQuiz').statsQuizPartial, 'full'),
        translations: () => relPartial(quizTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(quizTranslationPartial, 'list'),
    }
}