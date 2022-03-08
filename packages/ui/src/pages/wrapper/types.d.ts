import { Session } from "types";

export interface PageProps {
    title?: string;
    sessionChecked: boolean;
    redirect?: string;
    session: Session;
    restrictedToRoles?: string[];
    children: JSX.Element;
}

export interface RunRoutinePageProps {
    session: Session;
}