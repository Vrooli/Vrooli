import { ChatMessage, ChatMessageCreateInput, ChatMessageTranslation, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput, ChatMessageUpdateInput, ChatMessageYou, ReactionSummary, User } from "@local/shared";
import { ShapeModel } from "types";
import { ChatShape } from "./chat";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ChatMessageTranslationShape = Pick<ChatMessageTranslation, "id" | "language" | "text"> & {
    __typename?: "ChatMessageTranslation";
}

export type ChatMessageShape = Pick<ChatMessage, "id"> & {
    __typename: "ChatMessage";
    created_at: string; // Only used by the UI
    updated_at: string; // Only used by the UI
    chat?: { id: string } | ChatShape;
    isUnsent?: boolean; // Only used by the UI
    versionOfId?: string;
    parent?: { id: string } | ChatMessageShape | null;
    reactionSummaries: ReactionSummary[]; // Only used by the UI
    translations: ChatMessageTranslationShape[];
    user?: Partial<User> & { id: string };
    you?: ChatMessageYou; // Only used by the UI
}

export const shapeChatMessageTranslation: ShapeModel<ChatMessageTranslationShape, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text"), a),
};

export const shapeChatMessage: ShapeModel<ChatMessageShape, ChatMessageCreateInput, ChatMessageUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "versionOfId"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeChatMessageTranslation),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatMessageTranslation),
    }, a),
};
