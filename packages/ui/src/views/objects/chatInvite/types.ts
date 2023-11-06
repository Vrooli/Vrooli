import { ChatInvite } from "@local/shared";
import { FormProps } from "forms/types";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { UpsertProps } from "../types";

export type ChatInvitesUpsertProps = Omit<UpsertProps<ChatInvite[]>, "overrideObject"> & {
    invites: ChatInviteShape[];
}
export type ChatInvitesFormProps = FormProps<ChatInvite[], ChatInviteShape[]>
