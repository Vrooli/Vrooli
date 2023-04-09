export interface PageProps {
    excludePageContainer?: boolean;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    sx?: { [key: string]: any };
    children: JSX.Element;
}