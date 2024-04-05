import { Dialog, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import { UpTransition } from "../../transitions";
import { LargeDialogProps, MaybeLargeDialogProps } from "../types";

export const LargeDialog = ({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    sxs,
}: LargeDialogProps) => {
    const { palette, spacing } = useTheme();
    const [zIndex, handleTransitionExit] = useZIndex(isOpen, true, 1000);

    return (
        <Dialog
            id={id}
            open={isOpen}
            onClose={onClose}
            onTransitionExited={handleTransitionExit}
            scroll="paper"
            aria-labelledby={titleId}
            TransitionComponent={UpTransition}
            sx={{
                zIndex,
                ...sxs?.root,
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex,
                        margin: { xs: 0, sm: 2, md: 4 },
                        minWidth: { xs: "100vw", sm: "unset" },
                        maxWidth: { xs: "100vw", sm: "calc(100vw - 64px)" },
                        bottom: { xs: 0, sm: "auto" },
                        paddingBottom: "env(safe-area-inset-bottom)",
                        top: { xs: "auto", sm: undefined },
                        position: { xs: "absolute", sm: "relative" },
                        display: { xs: "block", sm: "inline-block" },
                        borderRadius: { xs: `${spacing(1)} ${spacing(1)} 0 0`, sm: 2 },
                        background: palette.mode === "light" ? "#c2cadd" : palette.background.default,
                        color: palette.background.textPrimary,
                        "& > .MuiDialogContent-root": {
                            position: "relative",
                        },
                        ...sxs?.paper,
                    },
                },
            }}
        >
            {children}
        </Dialog>
    );
};

/** Wraps children in a dialog is display is dialog */
export const MaybeLargeDialog = ({
    children,
    display,
    isOpen,
    onClose,
    ...props
}: MaybeLargeDialogProps) => {
    return display === "dialog" ? (
        <LargeDialog
            onClose={onClose ?? (() => {
                console.warn("onClose not passed to MaybeLargeDialog");
            })}
            isOpen={isOpen ?? false}
            {...props}
        >
            {children}
        </LargeDialog>
    ) : (
        <>{children}</>
    );
};
