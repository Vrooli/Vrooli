import { Quiz, QuizCreateInput, QuizUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizShape = Pick<Quiz, "id"> & {
    __typename: "Quiz";
}

export const shapeQuiz: ShapeModel<QuizShape, QuizCreateInput, QuizUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any,
};
