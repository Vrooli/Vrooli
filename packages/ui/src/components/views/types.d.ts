import { profile_profile } from "graphql/generated/profile";
import { DecisionStep, Node, Organization, Project, Routine, Session, Standard, User } from "types";

export interface CreateProps<T> {
    session: Session;
    onCreated: (item: T) => void;
    onCancel: () => void;
}
export interface UpdateProps<T> {
    session: Session;
    onUpdated: (item: T) => void;
    onCancel: () => void;
}
export interface ViewProps<T> {
    session: Session;
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
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

export interface StandardCreateProps extends CreateProps<Standard> {}
export interface StandardUpdateProps extends UpdateProps<Standard> {}
export interface StandardViewProps extends ViewProps<Standard> {}

export interface UserViewProps extends ViewProps<User> {}


export interface SettingsBaseProps {
    profile: profile_profile | undefined;
    onUpdated: (profile: profile) => void;
}
export interface SettingsAuthenticationProps extends SettingsBaseProps {
    session: Session
}
export interface SettingsDisplayProps extends SettingsBaseProps {
    session: Session;
}
export interface SettingsNotificationsProps extends SettingsBaseProps {
    session: Session
}
export interface SettingsProfileProps extends SettingsBaseProps {
    session: Session;
}

export interface SubroutineViewProps {
    loading: boolean;
    data: Routine | null;
    session: Session;
}

export interface DecisionViewProps {
    data: DecisionStep
    handleDecisionSelect: (node: Node) => void;
    nodes: Node[];
    session: Session;
}

export interface RunRoutineViewProps extends ViewProps<Routine> {
    handleClose: () => void; // View is always in a dialog
}