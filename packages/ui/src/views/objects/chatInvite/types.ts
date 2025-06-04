import { type ChatInvite, type ChatInviteShape } from "@local/shared";
import { type CrudPropsDialog, type FormProps } from "../../../types.js";

type ChatInvitesUpsertPropsDialog = Omit<CrudPropsDialog<ChatInvite[]>, "overrideObject"> & {
    invites: ChatInviteShape[];
    isMutate: boolean;
};
export type ChatInvitesUpsertProps = ChatInvitesUpsertPropsDialog; // Currently no page version
export type ChatInvitesFormProps = FormProps<ChatInvite[], ChatInviteShape[]> & Pick<ChatInvitesUpsertPropsDialog, "isMutate">;
