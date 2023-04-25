import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizQuestionShape = Pick<QuizQuestion, 'id'> & {
    __typename?: 'QuizQuestion';
}

export const shapeQuizQuestion: ShapeModel<QuizQuestionShape, QuizQuestionCreateInput, QuizQuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}