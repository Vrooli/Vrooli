import { ChatInvite } from "@local/shared";
import { FormProps } from "forms/types";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { CrudPropsDialog } from "../types";

type ChatInvitesUpsertPropsDialog = Omit<CrudPropsDialog<ChatInvite[]>, "overrideObject"> & {
    invites: ChatInviteShape[];
};
export type ChatInvitesUpsertProps = ChatInvitesUpsertPropsDialog; // Currently no page version
export type ChatInvitesFormProps = FormProps<ChatInvite[], ChatInviteShape[]>
