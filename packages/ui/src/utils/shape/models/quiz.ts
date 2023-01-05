import { Quiz, QuizCreateInput, QuizUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizShape = Pick<Quiz, 'id'>

export const shapeQuiz: ShapeModel<QuizShape, QuizCreateInput, QuizUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}