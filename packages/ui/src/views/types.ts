import { RoutineVersion } from "@shared/consts";
import React from "react";

/**
 * Views can be displayed as full pages or as dialogs
 */
export type ViewDisplayType = "dialog" | "page";

export type BaseViewProps = {
    display?: ViewDisplayType;
}

export interface ViewProps<T> extends BaseViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
    zIndex?: number;
}

export interface UpsertProps<T> extends BaseViewProps {
    isCreate: boolean;
    onCancel?: () => any;
    onCompleted?: (data: T) => any;
    zIndex?: number;
}

export interface AboutViewProps extends BaseViewProps { }

export interface AwardsViewProps extends BaseViewProps { }

export interface CalendarViewProps extends BaseViewProps { }

export interface HistorySearchViewProps extends BaseViewProps { }

export interface ObjectViewProps extends BaseViewProps { }

export interface PremiumViewProps extends BaseViewProps { }

export interface SearchViewProps extends BaseViewProps { }

export interface StartViewProps extends BaseViewProps { }

export interface StatsViewProps extends BaseViewProps { }

export interface ReportsViewProps extends BaseViewProps { }

export interface BuildViewProps extends ViewProps<RoutineVersion> {
    handleCancel: () => void;
    handleClose: () => void;
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

export interface CalendarViewProps extends BaseViewProps { }

export interface WelcomeViewProps extends BaseViewProps { };
