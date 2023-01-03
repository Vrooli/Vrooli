import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuestionAnswerShape = Pick<QuestionAnswer, 'id'>

export const shapeQuestionAnswer: ShapeModel<QuestionAnswerShape, QuestionAnswerCreateInput, QuestionAnswerUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}