import { MemberInvite } from "@local/shared";
import { UpsertProps } from "../types";
import { NewMemberInviteShape } from "./MemberInviteUpsert/MemberInviteUpsert";

export type MemberInviteUpsertProps = Omit<UpsertProps<MemberInvite>, "overrideObject"> & {
    invites: NewMemberInviteShape;
}

