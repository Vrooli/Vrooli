import { Session } from "@shared/consts";

export interface CreateProps<T> {
    session: Session;
    zIndex?: number;
}
export interface UpdateProps<T> {
    session: Session;
    zIndex?: number;
}
export interface ViewProps<T> {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
    session: Session;
    zIndex?: number;
}