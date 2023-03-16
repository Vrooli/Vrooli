import { Question, QuestionCreateInput, QuestionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuestionShape = Pick<Question, 'id'> & {
    __typename?: 'Question';
}

export const shapeQuestion: ShapeModel<QuestionShape, QuestionCreateInput, QuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}