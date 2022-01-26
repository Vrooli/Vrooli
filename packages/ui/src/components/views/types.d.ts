import { Organization, Project, Routine, Session, Standard, User } from "types";

export interface ViewProps<T> {
    session: Session;
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
}

export interface ActorViewProps extends ViewProps<User> {}

export interface OrganizationViewProps extends ViewProps<Organization> {}

export interface ProjectViewProps extends ViewProps<Project> {}

export interface RoutineViewProps extends ViewProps<Routine> {}

export interface StandardViewProps extends ViewProps<Standard> {}