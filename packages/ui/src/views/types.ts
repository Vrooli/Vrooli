import { OrArray, RoutineVersion } from "@local/shared";
import { ReactNode } from "react";
import { PartialOrArrayWithType, SxType } from "types";
import { ListObject } from "utils/display/listTools";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { MemberInviteShape } from "utils/shape/models/memberInvite";

/**
 * Views can be displayed as full pages or as dialogs
 */
export type ViewDisplayType = "dialog" | "page";
export type ViewProps = {
    /** Treated as a dialog when provided */
    isOpen?: boolean;
    onClose?: () => unknown;
}
export type ObjectViewProps<T extends OrArray<ListObject>> = ViewProps & {
    /**
    * Data known about the object, which cannot be fetched from the server or cache. 
    * This should always and only be used for dialogs.
    * 
    * This means that passing this in will render the view as a dialog instead of a page.
    */
    overrideObject?: PartialOrArrayWithType<T>;
}
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
    organization: MemberInviteShape["organization"];
}
export type ParticipantManageViewProps = ViewProps & {
    chat: ChatInviteShape["chat"];
}
export type PremiumViewProps = ViewProps
export type ResetPasswordViewProps = ViewProps
export type SearchViewProps = ViewProps
export type SignupViewProps = ViewProps
export type StatsSiteViewProps = ViewProps
export interface StatsObjectViewProps<T extends ListObject> extends ViewProps {
    handleObjectUpdate: (object: T) => unknown;
    object: T | null | undefined;
}
export type ReportsViewProps = ViewProps
export interface BuildViewProps extends ViewProps {
    handleCancel: () => unknown;
    handleSubmit: (updatedRoutineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">) => unknown;
    isEditing: boolean;
    loading: boolean;
    routineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">;
    translationData: {
        language: string;
        setLanguage: (language: string) => unknown;
        handleAddLanguage: (language: string) => unknown;
        handleDeleteLanguage: (language: string) => unknown;
        languages: string[];
    };
}
export interface ErrorBoundaryProps {
    children: ReactNode;
}
