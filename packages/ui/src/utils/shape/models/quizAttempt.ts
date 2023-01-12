import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type QuizAttemptShape = Pick<QuizAttempt, 'id'>

export const shapeQuizAttempt: ShapeModel<QuizAttemptShape, QuizAttemptCreateInput, QuizAttemptUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}