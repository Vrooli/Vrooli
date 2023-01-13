import { Node, ProjectVersion, RoutineVersion, RunRoutine, Session, User } from "@shared/consts";
import { RelationshipsObject } from "components/inputs/types";
import React from "react";
import { DecisionStep } from "types";

export interface CreateProps<T> {
    onCancel: () => void;
    onCreated: (item: T) => void;
    session: Session;
    zIndex: number;
}
export interface UpdateProps<T> {
    onCancel: () => void;
    onUpdated: (item: T) => void;
    session: Session;
    zIndex: number;
}
export interface ViewProps<T> {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
    session: Session;
    zIndex: number;
}

export interface ReportsViewProps {
    session: Session;
}
export interface UserViewProps extends ViewProps<User> {}


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