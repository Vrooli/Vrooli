import { QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizQuestionResponseShape = Pick<QuizQuestionResponse, 'id'> & {
    __typename?: 'QuizQuestionResponse';
}

export const shapeQuizQuestionResponse: ShapeModel<QuizQuestionResponseShape, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}