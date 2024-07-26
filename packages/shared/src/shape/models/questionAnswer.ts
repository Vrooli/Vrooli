import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type QuestionAnswerShape = Pick<QuestionAnswer, "id"> & {
    __typename: "QuestionAnswer";
}

export const shapeQuestionAnswer: ShapeModel<QuestionAnswerShape, QuestionAnswerCreateInput, QuestionAnswerUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
