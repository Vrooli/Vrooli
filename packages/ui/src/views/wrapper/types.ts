import { Session } from "@shared/consts";

export interface PageProps {
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    session: Session | undefined;
    sx?: { [key: string]: any };
    children: JSX.Element;
}