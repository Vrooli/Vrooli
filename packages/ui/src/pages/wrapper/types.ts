import { Session } from "@shared/consts";

export interface PageProps {
    title?: string;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    session: Session;
    children: JSX.Element;
}