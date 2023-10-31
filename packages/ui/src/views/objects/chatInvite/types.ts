import { ChatInvite } from "@local/shared";
import { UpsertProps } from "../types";
import { NewChatInviteShape } from "./ChatInviteUpsert/ChatInviteUpsert";

export type ChatInviteUpsertProps = Omit<UpsertProps<ChatInvite[]>, "overrideObject"> & {
    invites: NewChatInviteShape[];
}

