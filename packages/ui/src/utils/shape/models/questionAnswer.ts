import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuestionAnswerShape = Pick<QuestionAnswer, 'id'>

export const shapeQuestionAnswer: ShapeModel<QuestionAnswerShape, QuestionAnswerCreateInput, QuestionAnswerUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}