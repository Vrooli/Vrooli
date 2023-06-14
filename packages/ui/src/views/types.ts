import { Chat, RoutineVersion } from "@local/shared";
import React from "react";
import { AssistantTask } from "types";
import { ViewProps } from "./objects/types";

/**
 * Views can be displayed as full pages or as dialogs
 */
export type ViewDisplayType = "dialog" | "page";

export type BaseViewProps = {
    display?: ViewDisplayType;
    onClose: () => void;
    zIndex: number;
}

export type AboutViewProps = BaseViewProps

export type AwardsViewProps = BaseViewProps

export type CalendarViewProps = BaseViewProps

export type ChatViewProps = BaseViewProps & {
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
    chatInfo: Partial<Chat>;
    context?: string | null | undefined;
    task?: AssistantTask;
    zIndex: number;
}

export type HistorySearchViewProps = BaseViewProps

export type MemberManageViewProps = BaseViewProps & {
    organizationId: string;
}

export type ObjectViewProps = BaseViewProps

export type PremiumViewProps = BaseViewProps

export type SearchViewProps = BaseViewProps

export type StartViewProps = BaseViewProps

export type StatsViewProps = BaseViewProps

export type ReportsViewProps = BaseViewProps

export interface BuildViewProps extends ViewProps<RoutineVersion> {
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
    children: React.ReactNode;
}

export type WelcomeViewProps = BaseViewProps;
