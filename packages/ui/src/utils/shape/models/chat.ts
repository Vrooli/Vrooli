import { Chat, ChatCreateInput, ChatMessage, ChatParticipant, ChatTranslation, ChatTranslationCreateInput, ChatTranslationUpdateInput, ChatUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ChatInviteShape, shapeChatInvite } from "./chatInvite";
import { LabelShape, shapeLabel } from "./label";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ChatTranslationShape = Pick<ChatTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ChatTranslation";
}

export type ChatShape = Pick<Chat, "id" | "openToAnyoneWithInvite"> & {
    __typename: "Chat";
    invites?: ChatInviteShape[] | null;
    labels?: ({ id: string } | LabelShape)[];
    messages: ChatMessage[]; // Ignored, but needed for ChatView
    organization?: { id: string } | null;
    participants: ChatParticipant[]; // Ignored, but needed for ChatView
    participantsDelete?: { id: string }[] | null;
    translations?: ChatTranslationShape[] | null;
}

export const shapeChatTranslation: ShapeModel<ChatTranslationShape, ChatTranslationCreateInput, ChatTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name"), a),
};

export const shapeChat: ShapeModel<ChatShape, ChatCreateInput, ChatUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "openToAnyoneWithInvite"),
        ...createRel(d, "invites", ["Create"], "many", shapeChatInvite),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "translations", ["Create"], "many", shapeChatTranslation),
        ...(d.organization ? { organizationConnect: d.organization.id } : {}),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeChatInvite),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatTranslation),
        ...(u.participantsDelete ? { participantsDelete: u.participantsDelete.map(m => m.id) } : {}),
    }, a),
};
