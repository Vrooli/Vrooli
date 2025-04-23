import { forwardRef } from "react";
import { ViewDisplayType } from "../../types.js";
import { randomString } from "../../utils/codes.js";
import { DialogTitle } from "../dialogs/DialogTitle/DialogTitle.js";
import { Navbar } from "./Navbar.js";
import { PartialNavbar } from "./PartialNavbar.js";
import { TopBarProps } from "./types.js";

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
            {...titleData}
        />
    );
    if (display === ViewDisplayType.Partial) return (
        <PartialNavbar
            {...titleData}
        />
    );
    return (
        <Navbar
            titleBehavior={titleBehavior}
            {...titleData}
        />
    );
});
TopBar.displayName = "TopBar";
