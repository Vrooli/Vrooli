import { MemberInvite } from "@local/shared";
import { FormProps } from "forms/types";
import { MemberInviteShape } from "utils/shape/models/memberInvite";
import { UpsertProps } from "../types";

export type MemberInvitesUpsertProps = Omit<UpsertProps<MemberInvite[]>, "overrideObject"> & {
    invites: MemberInviteShape[];
}
export type MemberInvitesFormProps = FormProps<MemberInvite[], MemberInviteShape[]>
