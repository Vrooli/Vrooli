import { Organization, Project, Routine, Standard, User } from "types";

export interface ActorViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<User>;
}

export interface OrganizationViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Organization>;
}

export interface ProjectViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Project>;
}

export interface RoutineViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Routine>;
}

export interface StandardViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Standard>;
}