import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type QuizAttemptShape = Pick<QuizAttempt, "id"> & {
    __typename: "QuizAttempt";
}

export const shapeQuizAttempt: ShapeModel<QuizAttemptShape, QuizAttemptCreateInput, QuizAttemptUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
