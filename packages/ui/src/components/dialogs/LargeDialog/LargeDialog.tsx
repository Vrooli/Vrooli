import { Box, Dialog, Popover, styled, useTheme } from "@mui/material";
import { useMemo } from "react";
import { Z_INDEX } from "../../../utils/consts.js";
import { UpTransition } from "../../transitions/UpTransition/UpTransition.js";
import { LargeDialogProps, MaybeLargeDialogProps } from "../types.js";

const defaultAnchorOrigin = {
    vertical: "bottom" as const,
    horizontal: "right" as const,
} as const;
const defaultTransformOrigin = {
    vertical: "top" as const,
    horizontal: "right" as const,
} as const;

export function LargeDialog({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    sxs,
    anchorEl,
    anchorOrigin,
    transformOrigin,
}: LargeDialogProps) {
    const { palette, spacing } = useTheme();
    const shadowColor = palette.mode === "dark" ? "230, 230, 230" : "0, 0, 0";

    const style = useMemo(function styleMemo() {
        return {
            zIndex: Z_INDEX.Dialog,
            ...sxs?.root,
            "& > .MuiDialog-container": {
                "& > .MuiPaper-root": {
                    zIndex: Z_INDEX.Dialog,
                    margin: { xs: 0, sm: 2, md: 4 },
                    minWidth: { xs: "100vw", sm: "50%" },
                    maxWidth: { xs: "100vw", sm: "calc(100vw - 64px)" },
                    height: "100%",
                    overflow: "hidden",
                    bottom: { xs: 0, sm: "auto" },
                    paddingBottom: "env(safe-area-inset-bottom)",
                    top: { xs: "auto", sm: undefined },
                    position: { xs: "absolute", sm: "relative" },
                    display: { xs: "block", sm: "inline-block" },
                    // eslint-disable-next-line no-magic-numbers
                    borderRadius: { xs: `${spacing(3)} ${spacing(3)} 0 0`, sm: 3 },
                    background: palette.background.default,
                    color: palette.background.textPrimary,
                    boxShadow: `0px 11px 15px -7px rgba(${shadowColor},0.2),0px 24px 38px 3px rgba(${shadowColor},0.14),0px 9px 46px 8px rgba(${shadowColor},0.12)`,
                    "& > .MuiDialogContent-root": {
                        position: "relative",
                        tabIndex: 0,
                        outline: "none",
                        "&:focus": {
                            outline: "none",
                        },
                        "&:focus-visible": {
                            outline: `2px solid ${palette.primary.main}`,
                            outlineOffset: "-2px",
                        },
                        ...sxs?.content,
                    },
                    "& > .MuiBox-root": {
                        "& > .MuiDialogTitle-root": {
                            paddingTop: 0,
                            paddingBottom: 0,
                        },
                    },
                    ...sxs?.paper,
                },
            },
        } as const;
    }, [palette.background.default, palette.background.textPrimary, palette.primary.main, shadowColor, spacing, sxs?.content, sxs?.paper, sxs?.root]);

    // If anchorEl is provided, use Popover instead of Dialog
    if (anchorEl) {
        return (
            <Popover
                id={id}
                open={isOpen}
                anchorEl={anchorEl}
                onClose={onClose}
                anchorOrigin={anchorOrigin ?? defaultAnchorOrigin}
                transformOrigin={transformOrigin ?? defaultTransformOrigin}
                sx={style}
            >
                {children}
            </Popover>
        );
    }

    return (
        <Dialog
            id={id}
            open={isOpen}
            onClose={onClose}
            scroll="paper"
            aria-labelledby={titleId}
            TransitionComponent={UpTransition}
            sx={style}
        >
            {children}
        </Dialog>
    );
}

function fallbackOnClose() {
    console.warn("onClose not passed to MaybeLargeDialog");
}

const NotDialogBox = styled(Box)(({ theme }) => ({
    background: theme.palette.background.default,
    height: "100vh",
    overflowY: "auto",
    overflowX: "hidden",
    position: "relative",
    tabIndex: 0,
    role: "region",
    "&:focus": {
        outline: "none",
    },
    "&:focus-visible": {
        outline: "2px solid",
        outlineOffset: "-2px",
    },
}));

/** Wraps children in a dialog is display is dialog */
export function MaybeLargeDialog({
    children,
    display,
    isOpen,
    onClose,
    ...props
}: MaybeLargeDialogProps) {
    return display === "dialog" ? (
        <LargeDialog
            onClose={onClose ?? fallbackOnClose}
            isOpen={isOpen ?? false}
            {...props}
        >
            {children}
        </LargeDialog>
    ) : (
        <NotDialogBox>
            {children}
        </NotDialogBox>
    );
}
