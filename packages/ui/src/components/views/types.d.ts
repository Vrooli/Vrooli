import { RelationshipsObject } from "components/inputs/types";
import React from "react";
import { DecisionStep, Node, Profile, Routine, Run, Session, User } from "types";

interface CreateProps<T> {
    onCancel: () => void;
    onCreated: (item: T) => void;
    session: Session;
    zIndex: number;
}
interface UpdateProps<T> {
    onCancel: () => void;
    onUpdated: (item: T) => void;
    session: Session;
    zIndex: number;
}
interface ViewProps<T> {
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
    profile: Profile | undefined;
    onUpdated: (profile: Profile) => void;
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
    owner: Routine['owner'] | null | undefined;
    routine: Routine | null | undefined;
    run: Run | null | undefined;
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

export interface RunViewProps extends ViewProps<Routine> {
    handleClose: () => void;
    routine: Routine;
}

export interface BuildViewProps extends ViewProps<Routine> {
    handleCancel: () => void;
    handleClose: () => void;
    handleSubmit: (updatedRoutine: Pick<Routine, 'id' | 'nodes' | 'nodeLinks'>) => void;
    isEditing: boolean;
    loading: boolean;
    owner: RelationshipsObject['owner'] | null;
    routine: Pick<Routine, 'id' | 'nodes' | 'nodeLinks'>;
    translationData: {
        language: string;
        setLanguage: (language: string) => void;
        handleAddLanguage: (language: string) => void;
        handleDeleteLanguage: (language: string) => void;
        translations: Routine['translations'];
    };
}

export interface ErrorBoundaryProps {
    children: React.ReactNode;
}