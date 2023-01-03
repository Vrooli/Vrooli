import { Quiz, QuizCreateInput, QuizUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuizShape = Pick<Quiz, 'id'>

export const shapeQuiz: ShapeModel<QuizShape, QuizCreateInput, QuizUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}