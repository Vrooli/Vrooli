import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle";
import { forwardRef } from "react";
import { Navbar } from "../Navbar/Navbar";
import { TopBarProps } from "../types";

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = forwardRef(({
    below,
    display,
    hideTitleOnDesktop,
    onClose,
    ...titleData
}: TopBarProps, ref) => {

    if (display === "dialog") return (
        <DialogTitle
            ref={ref}
            below={below}
            id={titleData?.titleId ?? Math.random().toString(36).substring(2, 15)}
            onClose={onClose}
            {...titleData}
        />
    );
    return (
        <Navbar
            ref={ref}
            below={below}
            shouldHideTitle={hideTitleOnDesktop}
            {...titleData}
        />
    );
});
