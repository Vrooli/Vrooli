import { ChatInvite, ChatInviteCreateInput, ChatInviteStatus, ChatInviteUpdateInput, ChatInviteYou } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type ChatInviteShape = Pick<ChatInvite, "id" | "message"> & {
    __typename: "ChatInvite";
    created_at: string; // Only used by the UI
    updated_at: string; // Only used by the UI
    status: ChatInviteStatus; // Ignored when mutating, so don't get any ideas
    chat: { __typename: "Chat", id: string };
    user: { __typename: "User", id: string };
    you?: ChatInviteYou; // Only used by the UI
}

export const shapeChatInvite: ShapeModel<ChatInviteShape, ChatInviteCreateInput, ChatInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
    }, a),
};
