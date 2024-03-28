import { ChatMessage, ChatMessageCreateInput, ChatMessageParent, ChatMessageTranslation, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput, ChatMessageUpdateInput, ChatMessageYou, ReactionSummary, User } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ChatShape } from "./chat";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ChatMessageTranslationShape = Pick<ChatMessageTranslation, "id" | "language" | "text"> & {
    __typename?: "ChatMessageTranslation";
}

export type ChatMessageStatus = "unsent" | "editing" | "sending" | "sent" | "failed";
export type ChatMessageShape = Pick<ChatMessage, "id" | "versionIndex"> & {
    __typename: "ChatMessage";
    created_at: string; // Only used by the UI
    updated_at: string; // Only used by the UI
    chat?: CanConnect<ChatShape> | null;
    /** If not provided, we'll assume it's sent */
    status?: ChatMessageStatus;
    parent?: CanConnect<ChatMessageParent> | null;
    reactionSummaries: ReactionSummary[]; // Only used by the UI
    translations: ChatMessageTranslationShape[];
    user?: CanConnect<User> | null;
    you?: ChatMessageYou; // Only used by the UI
}

export const shapeChatMessageTranslation: ShapeModel<ChatMessageTranslationShape, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text"), a),
};

export const shapeChatMessage: ShapeModel<ChatMessageShape, ChatMessageCreateInput, ChatMessageUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "versionIndex"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeChatMessageTranslation),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatMessageTranslation),
    }, a),
};
