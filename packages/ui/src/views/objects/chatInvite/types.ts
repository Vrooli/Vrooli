import { ChatInvite } from "@local/shared";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { UpsertProps } from "../types";

export type ChatInviteUpsertProps = Omit<UpsertProps<ChatInvite[]>, "overrideObject"> & {
    invites: ChatInviteShape[];
}

