import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle";
import { forwardRef } from "react";
import { getTranslatedTitleAndHelp } from "utils/display/translationTools";
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
    const title = getTranslatedTitleAndHelp(titleData)?.title;
    const help = getTranslatedTitleAndHelp(titleData)?.help;

    if (display === "dialog") return (
        <DialogTitle
            ref={ref}
            below={below}
            id={titleData?.titleId ?? Math.random().toString(36).substring(2, 15)}
            title={title}
            onClose={onClose}
        />
    );
    return (
        <Navbar
            ref={ref}
            below={below}
            help={help}
            shouldHideTitle={hideTitleOnDesktop}
            title={title}
        />
    );
});
