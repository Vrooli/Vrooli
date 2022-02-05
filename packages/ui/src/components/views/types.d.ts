import { Organization, Project, Routine, Session, Standard, User } from "types";

export interface AddProps<T> {
    session: Session;
    onAdded: (item: T) => void;
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

export interface OrganizationAddProps extends AddProps<Organization> {}
export interface OrganizationUpdateProps extends UpdateProps<Organization> {}
export interface OrganizationViewProps extends ViewProps<Organization> {}

export interface ProjectAddProps extends AddProps<Project> {}
export interface ProjectUpdateProps extends UpdateProps<Project> {}
export interface ProjectViewProps extends ViewProps<Project> {}

export interface RoutineAddProps extends AddProps<Routine> {}
export interface RoutineUpdateProps extends UpdateProps<Routine> {}
export interface RoutineViewProps extends ViewProps<Routine> {}

export interface StandardAddProps extends AddProps<Standard> {}
export interface StandardUpdateProps extends UpdateProps<Standard> {}
export interface StandardViewProps extends ViewProps<Standard> {}

export interface UserUpdateProps extends UpdateProps<User> {}
export interface UserViewProps extends ViewProps<User> {}