import { Question, QuestionCreateInput, QuestionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuestionShape = Pick<Question, 'id'>

export const shapeQuestion: ShapeModel<QuestionShape, QuestionCreateInput, QuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}