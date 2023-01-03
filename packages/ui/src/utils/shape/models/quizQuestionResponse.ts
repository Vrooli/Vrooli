import { QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuizQuestionResponseShape = Pick<QuizQuestionResponse, 'id'>

export const shapeQuizQuestionResponse: ShapeModel<QuizQuestionResponseShape, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}