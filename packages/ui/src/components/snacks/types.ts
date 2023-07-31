import { SnackSeverity } from "./BasicSnack/BasicSnack";

export interface BasicSnackProps {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => unknown;
    buttonText?: string;
    /** Anything you'd like to log in development mode */
    data?: unknown;
    handleClose: () => unknown;
    id: string;
    message?: string;
    severity?: SnackSeverity | `${SnackSeverity}`;
}

export interface CookiesSnackProps {
    handleClose: () => unknown;
}
