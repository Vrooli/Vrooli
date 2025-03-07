import { MemberInvite, MemberInviteShape } from "@local/shared";
import { CrudPropsDialog, FormProps } from "../../../types.js";

type MemberInvitesUpsertPropsDialog = Omit<CrudPropsDialog<MemberInvite[]>, "overrideObject"> & {
    invites: MemberInviteShape[];
    isMutate: boolean;
};
export type MemberInvitesUpsertProps = MemberInvitesUpsertPropsDialog; // Currently no page version
export type MemberInvitesFormProps = FormProps<MemberInvite[], MemberInviteShape[]> & Pick<MemberInvitesUpsertPropsDialog, "isMutate">;
