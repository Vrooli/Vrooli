import { ChatInvite, ChatInviteCreateInput, ChatInviteUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ChatInviteShape = Pick<ChatInvite, "id" | "message"> & {
    __typename?: "ChatInvite";
    chat: { id: string };
    user: { id: string };
}

export const shapeChatInvite: ShapeModel<ChatInviteShape, ChatInviteCreateInput, ChatInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
        ...updateRel(o, u, "chat", ["Connect"], "one"),
        ...updateRel(o, u, "user", ["Connect"], "one"),
    }, a),
};
