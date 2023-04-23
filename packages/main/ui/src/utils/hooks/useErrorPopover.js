import { jsx as _jsx } from "react/jsx-runtime";
import { exists, uppercaseFirstLetter } from "@local/utils";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo, useState } from "react";
import { PopoverWithArrow } from "../../components/dialogs/PopoverWithArrow/PopoverWithArrow";
export const useErrorPopover = ({ errors, onSetSubmitting }) => {
    const [errorAnchorEl, setErrorAnchorEl] = useState(null);
    const openPopover = useCallback((ev) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target);
    }, []);
    const closePopover = useCallback(() => {
        setErrorAnchorEl(null);
        if (typeof onSetSubmitting === "function") {
            onSetSubmitting(false);
        }
    }, [onSetSubmitting]);
    const errorMessage = useMemo(() => {
        const filteredErrors = Object.entries(errors ?? {}).filter(([key, value]) => exists(value));
        const toListItem = (str, level) => { return `${"  ".repeat(level)}* ${str}`; };
        const errorList = filteredErrors.map(([key, value]) => {
            if (Array.isArray(value)) {
                return toListItem(uppercaseFirstLetter(key), 0) + ": \n" + value.map((str) => toListItem(str, 1)).join("\n");
            }
            else {
                return toListItem(uppercaseFirstLetter(key + ": " + value), 0);
            }
        }).join("\n");
        return errorList;
    }, [errors]);
    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const Popover = useCallback(() => {
        return (_jsx(PopoverWithArrow, { anchorEl: errorAnchorEl, handleClose: closePopover, sxs: {
                root: {
                    "& ul": {
                        paddingInlineStart: "20px",
                        margin: "8px",
                    },
                },
            }, children: _jsx(Markdown, { children: errorMessage }) }));
    }, [closePopover, errorAnchorEl, errorMessage]);
    return { errorMessage, openPopover, closePopover, hasErrors, Popover };
};
//# sourceMappingURL=useErrorPopover.js.map