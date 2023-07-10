import { SxType } from "types";

export interface PageProps {
    children: JSX.Element;
    excludePageContainer?: boolean;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    sx?: SxType;
}
