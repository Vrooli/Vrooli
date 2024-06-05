import { ListObject, OrArray } from "@local/shared";
import { ReactNode } from "react";
import { PartialOrArrayWithType, SxType } from "types";
import { ChatShape } from "utils/shape/models/chat";
import { MemberInviteShape } from "utils/shape/models/memberInvite";

/** Views can be displayed as full pages or as dialogs */
export type ViewDisplayType = "dialog" | "page" | "partial";
export type ViewPropsBase = {
    display: "dialog" | "page" | "partial";
};
export type ViewPropsPartial = ViewPropsBase & {
    display: "partial";
    isOpen?: never;
    onClose?: never;
}
export type ViewPropsPage = ViewPropsBase & {
    display: "page";
    isOpen?: never;
    onClose?: never;
}
export type ViewPropsDialog = ViewPropsBase & {
    display: "dialog";
    isOpen: boolean;
    onClose: () => unknown;
}
export type ViewProps = ViewPropsPartial | ViewPropsPage | ViewPropsDialog;

export type ObjectViewPropsBase = ViewProps;
export type ObjectViewPropsPage = ObjectViewPropsBase & {
    overrideObject?: never;
}
export type ObjectViewPropsDialog<T extends OrArray<ListObject>> = ObjectViewPropsBase & {
    /**  Data known about the object, which cannot be fetched from the server or cache. */
    overrideObject?: PartialOrArrayWithType<T>;
}
export type ObjectViewPropsPartial<T extends OrArray<ListObject>> = ObjectViewPropsBase & {
    /**  Data known about the object, which cannot be fetched from the server or cache. */
    overrideObject?: PartialOrArrayWithType<T>;
}
export type ObjectViewProps<T extends OrArray<ListObject>> = ObjectViewPropsPage | ObjectViewPropsDialog<T> | ObjectViewPropsPartial<T>;
export interface PageProps {
    children: JSX.Element;
    excludePageContainer?: boolean;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    sx?: SxType;
}

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
