import { Session } from "@shared/consts";

export interface PageProps {
    title?: string;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    session: Session;
    sx?: { [key: string]: any };
    children: JSX.Element;
}