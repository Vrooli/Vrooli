import { QuizAttemptYou } from "@shared/consts";
import { GqlPartial } from "types";

export const quizAttemptYouPartial: GqlPartial<QuizAttemptYou> = {
    __typename: 'QuizAttemptYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
    }),
}

export const quizAttemptFields = ['QuizAttempt', `{
    id
}`] as const;