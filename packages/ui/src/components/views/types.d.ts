import { DecisionStep, Node, Organization, Profile, Project, Routine, Run, Session, Standard, User } from "types";

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

export interface OrganizationCreateProps extends CreateProps<Organization> {}
export interface OrganizationUpdateProps extends UpdateProps<Organization> {}
export interface OrganizationViewProps extends ViewProps<Organization> {}

export interface ProjectCreateProps extends CreateProps<Project> {}
export interface ProjectUpdateProps extends UpdateProps<Project> {}
export interface ProjectViewProps extends ViewProps<Project> {}

export interface RoutineCreateProps extends CreateProps<Routine> {}
export interface RoutineUpdateProps extends UpdateProps<Routine> {}
export interface RoutineViewProps extends ViewProps<Routine> {}

export interface StandardCreateProps extends CreateProps<Standard> {
    session: Session;
}
export interface StandardUpdateProps extends UpdateProps<Standard> {}
export interface StandardViewProps extends ViewProps<Standard> {}

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
    handleClose: (wasModified: boolean) => void;
    loading: boolean;
    onChange: (routine: Routine) => void;
    routine: Routine | null;
    zIndex: number;
}