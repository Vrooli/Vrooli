import { QuestionTranslation } from "@shared/consts";
import { GqlPartial } from "types";

export const questionTranslationPartial: GqlPartial<QuestionTranslation> = {
    __typename: 'QuestionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const listQuestionFields = ['Question', `{
    id
}`] as const;
export const questionFields = ['Question', `{
    id
}`] as const;