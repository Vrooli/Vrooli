import { exists, uppercaseFirstLetter } from "@local/shared";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useCallback, useMemo, useState } from "react";
import { FormErrors } from "types";

interface UsePopoverMenuOptions {
    errors: FormErrors | undefined;
    onSetSubmitting?: (isSubmitting: boolean) => unknown;
}

interface UsePopoverMenuReturn {
    closePopover: () => unknown;
    errorMessage: string;
    hasErrors: boolean;
    openPopover: (event: React.MouseEvent | React.TouchEvent) => unknown;
    Popover: () => JSX.Element;
}

export function useErrorPopover({
    errors,
    onSetSubmitting,
}: UsePopoverMenuOptions): UsePopoverMenuReturn {
    // Errors popup
    const [errorAnchorEl, setErrorAnchorEl] = useState<any | null>(null);
    const openPopover = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target);
    }, []);
    const closePopover = useCallback(() => {
        setErrorAnchorEl(null);
        if (typeof onSetSubmitting === "function") {
            onSetSubmitting(false);
        }
    }, [onSetSubmitting]);

    // Errors as a markdown list
    const errorMessage = useMemo<string>(() => {
        // Filter out null and undefined errors
        const filteredErrors = Object.entries(errors ?? {}).filter(([, value]) => exists(value)) as [string, string | string[]][];
        // If there is an error of key "_form", return that error instead of a list
        const formError = filteredErrors.find(([key]) => key === "_form");
        if (formError) return uppercaseFirstLetter(formError[1] as string);
        // Helper to convert string to markdown list item
        const toListItem = (str: string, level: number) => { return `${"  ".repeat(level)}* ${str}`; };
        // Convert errors to markdown list
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
        return (
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closePopover}
                sxs={{
                    root: {
                        // Remove horizontal spacing for list items
                        "& ul": {
                            paddingInlineStart: "20px",
                            margin: "8px",
                        },
                    },
                }}
            >
                <MarkdownDisplay content={errorMessage} />
            </PopoverWithArrow>
        );
    }, [closePopover, errorAnchorEl, errorMessage]);

    return { errorMessage, openPopover, closePopover, hasErrors, Popover };
}
