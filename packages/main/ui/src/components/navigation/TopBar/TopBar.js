import { jsx as _jsx } from "react/jsx-runtime";
import { forwardRef } from "react";
import { getTranslatedTitleAndHelp } from "../../../utils/display/translationTools";
import { DialogTitle } from "../../dialogs/DialogTitle/DialogTitle";
import { Navbar } from "../Navbar/Navbar";
export const TopBar = forwardRef(({ below, display, onClose, titleData, }, ref) => {
    const title = getTranslatedTitleAndHelp(titleData)?.title;
    const help = getTranslatedTitleAndHelp(titleData)?.help;
    if (display === "dialog")
        return (_jsx(DialogTitle, { ref: ref, below: below, id: titleData?.titleId ?? Math.random().toString(36).substring(2, 15), title: title, onClose: onClose }));
    return (_jsx(Navbar, { ref: ref, below: below, help: help, shouldHideTitle: titleData?.hideOnDesktop, title: title }));
});
//# sourceMappingURL=TopBar.js.map