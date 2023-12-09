import { Question, QuestionCreateInput, QuestionForType, QuestionTranslation, QuestionTranslationCreateInput, QuestionTranslationUpdateInput, QuestionUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { QuestionAnswerShape } from "./questionAnswer";
import { TagShape, shapeTag } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type QuestionTranslationShape = Pick<QuestionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "QuestionTranslation";
}

export type QuestionShape = Pick<Question, "id" | "isPrivate"> & {
    __typename: "Question";
    acceptedAnswer?: CanConnect<QuestionAnswerShape> | null;
    forObject?: { __typename: QuestionForType | `${QuestionForType}`, id: string } | null;
    referencing?: string;
    tags?: CanConnect<TagShape, "tag">[] | null;
    translations?: QuestionTranslationShape[] | null;
}

export const shapeQuestionTranslation: ShapeModel<QuestionTranslationShape, QuestionTranslationCreateInput, QuestionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name"), a),
};

export const shapeQuestion: ShapeModel<QuestionShape, QuestionCreateInput, QuestionUpdateInput> = {
    create: (d) => ({
        forObjectConnect: d.forObject?.id ?? undefined,
        forObjectType: d.forObject?.__typename as any ?? undefined,
        ...createPrims(d, "id", "isPrivate", "referencing"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createRel(d, "translations", ["Create"], "many", shapeQuestionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateRel(o, u, "acceptedAnswer", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeQuestionTranslation),
    }, a),
};
