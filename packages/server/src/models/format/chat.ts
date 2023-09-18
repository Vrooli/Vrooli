import { ChatModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ChatFormat: Formatter<ChatModelLogic> = {
    gqlRelMap: {
        __typename: "Chat",
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    prismaRelMap: {
        __typename: "Chat",
        creator: "User",
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    joinMap: { labels: "label", restrictedToRoles: "role" },
    countFields: {
        participantsCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};
