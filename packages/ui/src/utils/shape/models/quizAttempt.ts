import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizAttemptShape = Pick<QuizAttempt, 'id'> & {
    __typename?: 'QuizAttempt';
}

export const shapeQuizAttempt: ShapeModel<QuizAttemptShape, QuizAttemptCreateInput, QuizAttemptUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}