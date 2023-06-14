import { ChatModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Chat" as const;
export const ChatFormat: Formatter<ChatModelLogic> = {
    gqlRelMap: {
        __typename,
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    prismaRelMap: {
        __typename,
        creator: "User",
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    joinMap: { labels: "label", participants: "user", restrictedToRoles: "role" },
    countFields: {
        participantsCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};
