import type React from "react";
import { type SnackSeverity } from "./BasicSnack/BasicSnack.js";

export interface BasicSnackProps {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: React.MouseEvent<HTMLButtonElement>) => unknown;
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

// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed buttonClicked event handler type from 'any' to React.MouseEvent<HTMLButtonElement>
