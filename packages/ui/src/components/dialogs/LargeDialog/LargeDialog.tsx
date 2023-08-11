import { Dialog, useTheme } from "@mui/material";
import { UpTransition } from "../../transitions";
import { LargeDialogProps, MaybeLargeDialogProps } from "../types";

export const LargeDialog = ({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    zIndex,
    sxs,
}: LargeDialogProps) => {
    const { palette } = useTheme();

    return (
        <Dialog
            id={id}
            open={isOpen}
            onClose={onClose}
            scroll="paper"
            aria-labelledby={titleId}
            TransitionComponent={UpTransition}
            sx={{
                zIndex,
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
                        borderRadius: 2,
                        background: palette.mode === "light" ? "#c2cadd" : palette.background.default,
                        color: palette.background.textPrimary,
                        "& > .MuiDialogContent-root": {
                            position: "relative",
                        },
                        ...(sxs?.paper ?? {}),
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
    onClose,
    ...props
}: MaybeLargeDialogProps) => {
    return display === "dialog" ? (
        <LargeDialog onClose={onClose ?? (() => { })} {...props}>
            {children}
        </LargeDialog>
    ) : (
        <>{children}</>
    );
};
