import { Quiz, QuizCreateInput, QuizUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type QuizShape = Pick<Quiz, "id"> & {
    __typename: "Quiz";
}

export const shapeQuiz: ShapeModel<QuizShape, QuizCreateInput, QuizUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
