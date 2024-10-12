import { Box, Dialog, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import { useMemo } from "react";
import { UpTransition } from "../../transitions/UpTransition/UpTransition";
import { LargeDialogProps, MaybeLargeDialogProps } from "../types";

const DEFAULT_Z_INDEX_OFFSET = 1000;

export function LargeDialog({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    sxs,
    zIndexOffset = DEFAULT_Z_INDEX_OFFSET,
}: LargeDialogProps) {
    const { palette, spacing } = useTheme();
    const [zIndex, handleTransitionExit] = useZIndex(isOpen, true, zIndexOffset);

    const style = useMemo(function styleMemo() {
        return {
            zIndex,
            ...sxs?.root,
            "& > .MuiDialog-container": {
                "& > .MuiPaper-root": {
                    zIndex,
                    margin: { xs: 0, sm: 2, md: 4 },
                    minWidth: { xs: "100vw", sm: "unset" },
                    maxWidth: { xs: "100vw", sm: "calc(100vw - 64px)" },
                    height: "100%",
                    overflow: "hidden",
                    bottom: { xs: 0, sm: "auto" },
                    paddingBottom: "env(safe-area-inset-bottom)",
                    top: { xs: "auto", sm: undefined },
                    position: { xs: "absolute", sm: "relative" },
                    display: { xs: "block", sm: "inline-block" },
                    borderRadius: { xs: `${spacing(1)} ${spacing(1)} 0 0`, sm: 2 },
                    background: palette.background.default,
                    color: palette.background.textPrimary,
                    "& > .MuiDialogContent-root": {
                        position: "relative",
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
    }, [palette, spacing, sxs, zIndex]);

    return (
        <Dialog
            id={id}
            open={isOpen}
            onClose={onClose}
            onTransitionExited={handleTransitionExit}
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

const notDialogBoxprops = {
    height: "100vh",
    overflowY: "auto",
    overflowX: "hidden",
} as const;

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
        <Box sx={notDialogBoxprops}>
            {children}
        </Box>
    );
}
