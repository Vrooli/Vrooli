import { ChatShape, ListObject, MemberInviteShape } from "@local/shared";
import { ReactNode } from "react";
import { ViewProps } from "types";

export type AboutViewProps = ViewProps
export type AwardsViewProps = ViewProps
export type CalendarViewProps = ViewProps
export type ForgotPasswordViewProps = ViewProps
export type HistorySearchViewProps = ViewProps
export type LoginViewProps = ViewProps
export type MemberManageViewProps = ViewProps & {
    team: MemberInviteShape["team"];
}
export type ParticipantManageViewProps = ViewProps & {
    chat: ChatShape;
}
export type PremiumViewProps = ViewProps
export type ResetPasswordViewProps = ViewProps
export type SearchViewProps = ViewProps
export type SearchVersionViewProps = ViewProps
export type SignupViewProps = ViewProps
export type StatsSiteViewProps = ViewProps
export type StatsObjectViewProps<T extends ListObject> = ViewProps & {
    handleObjectUpdate: (object: T) => unknown;
    object: T | null | undefined;
}
export type ReportsViewProps = ViewProps
export interface ErrorBoundaryProps {
    children: ReactNode;
}
