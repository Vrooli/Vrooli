import { Chat, ChatCreateInput, ChatTranslation, ChatTranslationCreateInput, ChatTranslationUpdateInput, ChatUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ChatInviteShape, shapeChatInvite } from "./chatInvite";
import { ChatMessageShape, shapeChatMessage } from "./chatMessage";
import { ChatParticipantShape } from "./chatParticipant";
import { LabelShape, shapeLabel } from "./label";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ChatTranslationShape = Pick<ChatTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ChatTranslation";
}

export type ChatShape = Pick<Chat, "id" | "openToAnyoneWithInvite"> & {
    __typename: "Chat";
    /** Invites for new participants */
    invites: ChatInviteShape[];
    labels?: CanConnect<LabelShape>[] | null;
    messages: ChatMessageShape[];
    /** Potentially non-exhaustive list of current participants. Ignored by shapers */
    participants?: ChatParticipantShape[];
    /** Participants being removed from the chst */
    participantsDelete?: { id: string }[] | null;
    team?: CanConnect<TeamShape> | null;
    translations?: ChatTranslationShape[] | null;
}

export const shapeChatTranslation: ShapeModel<ChatTranslationShape, ChatTranslationCreateInput, ChatTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};

export const shapeChat: ShapeModel<ChatShape, ChatCreateInput, ChatUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "openToAnyoneWithInvite");
        return {
            ...prims,
            ...createRel(d, "invites", ["Create"], "many", shapeChatInvite, (m) => ({ ...m, chat: { id: prims.id } })),
            ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
            ...createRel(d, "messages", ["Create"], "many", shapeChatMessage, (m) => {
                console.log("preshaping chat message", m, prims);
                return { ...m, chat: { id: prims.id } };
            }),
            ...createRel(d, "translations", ["Create"], "many", shapeChatTranslation),
            ...(d.team ? { teamConnect: d.team.id } : {}),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeChatInvite, (m, i) => ({ ...m, chat: { id: i.id } })),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "messages", ["Create", "Update", "Delete"], "many", shapeChatMessage, (m, i) => ({ ...m, chat: { id: i.id } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatTranslation),
        ...(u.participantsDelete?.length ? { participantsDelete: u.participantsDelete.map(m => m.id) } : {}),
    }),
};
