import { MemberInvite } from "@local/shared";
import { FormProps } from "forms/types";
import { MemberInviteShape } from "utils/shape/models/memberInvite";
import { UpsertProps } from "../types";
import { NewMemberInviteShape } from "./MemberInvitesUpsert/MemberInvitesUpsert";

export type MemberInvitesUpsertProps = Omit<UpsertProps<MemberInvite>, "overrideObject"> & {
    invites: NewMemberInviteShape;
}
export type MemberInvitesFormProps = FormProps<MemberInvite[], MemberInviteShape[]>
