import { RoutineVersion, Session } from "@shared/consts";
import { RelationshipsObject } from "components/inputs/types";
import React from "react";
import { ViewProps } from "./objects/types";

/**
 * Views can be displayed as full pages or as dialogs
 */
export type ViewDisplayType = 'dialog' | 'page';

export type BaseViewProps = {
    display?: ViewDisplayType;
    session: Session | undefined;
}

export interface AwardsViewProps extends BaseViewProps {}

export interface CalendarViewProps extends BaseViewProps {}

export interface HistorySearchViewProps extends BaseViewProps {}

export interface NotFoundViewProps extends BaseViewProps {}

export interface  ObjectViewProps extends BaseViewProps {}

export interface PremiumViewProps extends BaseViewProps {}

export interface SearchViewProps extends BaseViewProps {}

export interface StartViewProps extends BaseViewProps {}

export interface StatsViewProps extends BaseViewProps {}

export interface ReportsViewProps extends BaseViewProps {}

export interface BuildViewProps extends ViewProps<RoutineVersion> {
    handleCancel: () => void;
    handleClose: () => void;
    handleSubmit: (updatedRoutineVersion: Pick<RoutineVersion, 'id' | 'nodes' | 'nodeLinks'>) => void;
    isEditing: boolean;
    loading: boolean;
    owner: RelationshipsObject['owner'] | null;
    routineVersion: Pick<RoutineVersion, 'id' | 'nodes' | 'nodeLinks'>;
    translationData: {
        language: string;
        setLanguage: (language: string) => void;
        handleAddLanguage: (language: string) => void;
        handleDeleteLanguage: (language: string) => void;
        translations: RoutineVersion['translations'];
    };
}

export interface ErrorBoundaryProps {
    children: React.ReactNode;
}

export interface CalendarViewProps extends BaseViewProps {}