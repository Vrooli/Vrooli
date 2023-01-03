import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuizQuestionShape = Pick<QuizQuestion, 'id'>

export const shapeQuizQuestion: ShapeModel<QuizQuestionShape, QuizQuestionCreateInput, QuizQuestionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}