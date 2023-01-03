import { Question, QuestionCreateInput, QuestionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuestionShape = Pick<Question, 'id'>

export const shapeQuestion: ShapeModel<QuestionShape, QuestionCreateInput, QuestionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}