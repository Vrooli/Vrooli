import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle";
import { forwardRef } from "react";
import { randomString } from "utils/codes";
import { Navbar } from "../Navbar/Navbar";
import { TopBarProps } from "../types";

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = forwardRef(({
    display,
    sxsNavbar,
    titleBehaviorDesktop,
    titleBehaviorMobile,
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
            sxs={sxsNavbar as any}
            titleBehaviorDesktop={titleBehaviorDesktop}
            titleBehaviorMobile={titleBehaviorMobile}
            {...titleData}
        />
    );
});
TopBar.displayName = "TopBar";
