import { QuestionAnswerTranslation } from "@shared/consts";
import { GqlPartial } from "types";

export const questionAnswerTranslationPartial: GqlPartial<QuestionAnswerTranslation> = {
    __typename: 'QuestionAnswerTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
    }),
}

export const listQuestionAnswerFields = ['QuestionAnswer', `{
    id
}`] as const;
export const questionAnswerFields = ['QuestionAnswer', `{
    id
}`] as const;