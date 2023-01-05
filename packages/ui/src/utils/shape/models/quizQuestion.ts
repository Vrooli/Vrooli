import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizQuestionShape = Pick<QuizQuestion, 'id'>

export const shapeQuizQuestion: ShapeModel<QuizQuestionShape, QuizQuestionCreateInput, QuizQuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}