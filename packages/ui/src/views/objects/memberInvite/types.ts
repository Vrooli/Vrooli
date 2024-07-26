import { MemberInvite, MemberInviteShape } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudPropsDialog } from "../types";

type MemberInvitesUpsertPropsDialog = Omit<CrudPropsDialog<MemberInvite[]>, "overrideObject"> & {
    invites: MemberInviteShape[];
    isMutate: boolean;
};
export type MemberInvitesUpsertProps = MemberInvitesUpsertPropsDialog; // Currently no page version
export type MemberInvitesFormProps = FormProps<MemberInvite[], MemberInviteShape[]> & Pick<MemberInvitesUpsertPropsDialog, "isMutate">;
