import { Chat, RoutineVersion } from "@local/shared";
import { ReactNode } from "react";
import { AssistantTask, PartialWithType } from "types";
import { ListObject } from "utils/display/listTools";

/**
 * Views can be displayed as full pages or as dialogs
 */
export type ViewDisplayType = "dialog" | "page";

export type ViewProps = {
    /** Treated as a dialog when provided */
    isOpen?: boolean;
    onClose?: () => unknown;
    zIndex: number;
}

export type ObjectViewProps<T extends ListObject> = ViewProps & {
    /**
    * Data known about the object, which cannot be fetched from the server or cache. 
    * This should always and only be used for dialogs.
    * 
    * This means that passing this in will render the view as a dialog instead of a page.
    */
    overrideObject?: PartialWithType<T>;
}

export type AboutViewProps = ViewProps

export type AwardsViewProps = ViewProps

export type CalendarViewProps = ViewProps

export type ChatViewProps = ViewProps & {
    botSettings?: string | null | undefined;
    /** 
     * Info for finding an existing chat or starting a new one.
     * 
     * Pass an ID if you want to find an existing chat.
     * 
     * Pass `{ 
     *     invites: [{ id: "abc-123"}, { id: "345-678"}],
     *     //...other chat info like translations and labels (optional)
     * }` to start a new chat with the given users.
     * */
    chatInfo?: Partial<Chat>;
    context?: string | null | undefined;
    task?: AssistantTask;
    zIndex: number;
}

export type HistorySearchViewProps = ViewProps

export type MemberManageViewProps = ViewProps & {
    organizationId: string;
}

export type PremiumViewProps = ViewProps

export type SearchViewProps = ViewProps

export type StartViewProps = ViewProps

export type StatsViewProps = ViewProps

export type ReportsViewProps = ViewProps

export interface BuildViewProps extends ViewProps {
    handleCancel: () => void;
    handleSubmit: (updatedRoutineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">) => void;
    isEditing: boolean;
    loading: boolean;
    routineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">;
    translationData: {
        language: string;
        setLanguage: (language: string) => void;
        handleAddLanguage: (language: string) => void;
        handleDeleteLanguage: (language: string) => void;
        languages: string[];
    };
}

export interface ErrorBoundaryProps {
    children: ReactNode;
}
