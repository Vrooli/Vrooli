import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type QuizAttemptShape = Pick<QuizAttempt, 'id'>

export const shapeQuizAttempt: ShapeModel<QuizAttemptShape, QuizAttemptCreateInput, QuizAttemptUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}