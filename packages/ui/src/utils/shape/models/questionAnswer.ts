import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuestionAnswerShape = Pick<QuestionAnswer, "id"> & {
    __typename?: "QuestionAnswer";
}

export const shapeQuestionAnswer: ShapeModel<QuestionAnswerShape, QuestionAnswerCreateInput, QuestionAnswerUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any,
};
