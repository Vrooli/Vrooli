import { Node, ProjectVersion, RoutineVersion, RunRoutine, Session, User } from "@shared/consts";
import { CommonKey } from "@shared/translations";
import { RelationshipsObject } from "components/inputs/types";
import React from "react";
import { DecisionStep } from "types";
import { ViewProps } from "./objects/types";

/**
 * Defines data to properly render a view, either as a full page 
 * or as a dialog
 */
export interface ViewUI {
    /**
     * Data to render the navbar/appbar. Title + '|' + site name = tab title.
     * 
     * NOTE: On desktop, the navbar does not change. Instead, top content is 
     * displayed above main content. This behavior does not effect appbars (i.e. dialogs)
     */
    top?: {
        // Non-translated title (or translation from object)
        title?: string;
        // Translated title using i18n
        titleKey?: CommonKey;
        titleVariables?: { [key: string]: string };
        // Non-translated help message
        help?: string;
        // Translated help message using i18n
        helpKey?: CommonKey;
        helpVariables?: { [key: string]: string };
    },
    /**
     * Data to render the main content of the view
     */
    main: JSX.Element,
    /**
     * Data to render the bottom actions (NOT the bottom navbar!). This is often 
     * used for form submission buttons
     */
    actions?: {
        // Buttons to render
        buttons: JSX.Element[];
        // Display horizontally along bottom or vertically on right/left side of screen
        direction: 'horizontal' | 'vertical';
    }
}

export interface AwardsViewProps {
    session: Session;
}

export interface CalendarViewProps {
    session: Session;
}

export interface HistorySearchViewProps {
    session: Session;
}

export interface  ObjectViewProps {
    session: Session;
}

export interface PremiumViewProps {
    session: Session;
}

export interface SearchViewProps {
    session: Session;
}

export interface StartViewProps {
    session: Session;
}

export interface StatsViewProps {
    session: Session;
}

export interface WelcomeViewProps {
    session: Session;
}

export interface ReportsViewProps {
    session: Session;
}

interface SettingsBaseProps {
    profile: User | undefined;
    onUpdated: (profile: User) => void;
    session: Session;
    zIndex: number;
}
export interface SettingsAuthenticationProps extends SettingsBaseProps {}
export interface SettingsDisplayProps extends SettingsBaseProps {}
export interface SettingsNotificationsProps extends SettingsBaseProps {}
export interface SettingsProfileProps extends SettingsBaseProps {}

export interface SubroutineViewProps {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => void;
    handleSaveProgress: () => void;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion['root']['owner'] | null | undefined;
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
    session: Session;
    zIndex: number;
}

export interface DecisionViewProps {
    data: DecisionStep
    handleDecisionSelect: (node: Node) => void;
    nodes: Node[];
    session: Session;
    zIndex: number;
}

export interface RunViewProps extends ViewProps<RoutineVersion> {
    handleClose: () => void;
    runnableObject: ProjectVersion | RoutineVersion;
}

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

export interface CalendarViewProps {
    session: Session;
}