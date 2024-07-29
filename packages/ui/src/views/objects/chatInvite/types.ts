import { ChatInvite, ChatInviteShape } from "@local/shared";
import { CrudPropsDialog, FormProps } from "../../../types";

type ChatInvitesUpsertPropsDialog = Omit<CrudPropsDialog<ChatInvite[]>, "overrideObject"> & {
    invites: ChatInviteShape[];
    isMutate: boolean;
};
export type ChatInvitesUpsertProps = ChatInvitesUpsertPropsDialog; // Currently no page version
export type ChatInvitesFormProps = FormProps<ChatInvite[], ChatInviteShape[]> & Pick<ChatInvitesUpsertPropsDialog, "isMutate">;
