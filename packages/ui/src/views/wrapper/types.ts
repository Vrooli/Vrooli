import { Session } from "@shared/consts";
import { OptionalTranslation } from "types";

export interface PageProps {
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    session: Session | undefined;
    sx?: { [key: string]: any };
    titleData?: OptionalTranslation;
    children: JSX.Element;
}