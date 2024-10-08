import { QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type QuizQuestionResponseShape = Pick<QuizQuestionResponse, "id"> & {
    __typename: "QuizQuestionResponse";
}

export const shapeQuizQuestionResponse: ShapeModel<QuizQuestionResponseShape, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
