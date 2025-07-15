import { forwardRef } from "react";
import { ViewDisplayType } from "../../types.js";
import { randomString } from "../../utils/codes.js";
import { DialogTitle } from "../dialogs/DialogTitle/DialogTitle.js";
import { Navbar } from "./Navbar.js";
import { PartialNavbar } from "./PartialNavbar.js";
import { type TopBarProps } from "./types.js";

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = forwardRef(({
    display,
    titleBehavior,
    ...titleData
}: TopBarProps, ref) => {

    if (display === ViewDisplayType.Dialog) return (
        <DialogTitle
            ref={ref}
            id={titleData?.titleId ?? randomString()}
            data-testid="topbar-dialog"
            data-display-type={display}
            {...titleData}
        />
    );
    if (display === ViewDisplayType.Partial) return (
        <PartialNavbar
            data-testid="topbar-partial"
            data-display-type={display}
            {...titleData}
        />
    );
    return (
        <Navbar
            titleBehavior={titleBehavior}
            data-testid="topbar-page"
            data-display-type={display}
            {...titleData}
        />
    );
});
TopBar.displayName = "TopBar";
