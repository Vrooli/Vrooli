import { forwardRef } from "react";
import { randomString } from "../../utils/codes.js";
import { DialogTitle } from "../dialogs/DialogTitle/DialogTitle.js";
import { Navbar } from "./Navbar.js";
import { TopBarProps } from "./types.js";

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = forwardRef(({
    display,
    titleBehavior,
    ...titleData
}: TopBarProps, ref) => {

    if (display === "dialog") return (
        <DialogTitle
            ref={ref}
            id={titleData?.titleId ?? randomString()}
            {...titleData}
        />
    );
    return (
        <Navbar
            ref={ref}
            titleBehavior={titleBehavior}
            {...titleData}
        />
    );
});
TopBar.displayName = "TopBar";
