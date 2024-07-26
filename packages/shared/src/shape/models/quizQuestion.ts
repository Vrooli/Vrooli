import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type QuizQuestionShape = Pick<QuizQuestion, "id"> & {
    __typename: "QuizQuestion";
}

export const shapeQuizQuestion: ShapeModel<QuizQuestionShape, QuizQuestionCreateInput, QuizQuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
