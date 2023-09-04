import { ChatMessage, ChatMessageCreateInput, ChatMessageTranslation, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput, ChatMessageUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ChatShape } from "./chat";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ChatMessageTranslationShape = Pick<ChatMessageTranslation, "id" | "language" | "text"> & {
    __typename?: "ChatMessageTranslation";
}

export type ChatMessageShape = Pick<ChatMessage, "id"> & {
    __typename?: "ChatMessage";
    chat?: { id: string } | ChatShape;
    isFork: boolean;
    fork?: { id: string } | ChatMessageShape;
    translations?: ChatMessageTranslationShape[] | null;
    user?: { id: string };
}

export const shapeChatMessageTranslation: ShapeModel<ChatMessageTranslationShape, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text"), a),
};

export const shapeChatMessage: ShapeModel<ChatMessageShape, ChatMessageCreateInput, ChatMessageUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isFork"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "fork", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeChatMessageTranslation),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatMessageTranslation),
    }, a),
};
