import { Dialog, useTheme } from "@mui/material";
import { LargeDialogProps } from "../types";

export const LargeDialog = ({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    zIndex,
}: LargeDialogProps) => {
    const { breakpoints, palette } = useTheme();

    return (
        <Dialog
            id={id}
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleId}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    // Full width on small screens
                },
                '& .MuiPaper-root': {
                    margin: { xs: 0, sm: 2, md: 4 },
                    minWidth: `min(${breakpoints.values.sm}px, 100%)`,
                    maxWidth: { xs: '100vw', sm: 'calc(100vw - 64px)' },
                    bottom: { xs: 0, sm: 'auto' },
                    top: { xs: 'auto', sm: undefined },
                    position: { xs: 'absolute', sm: 'inherit' },
                    display: { xs: 'block', sm: 'inline-block' },
                    background: palette.background.default,
                    color: palette.background.textPrimary,
                },
                // Remove ::after element that is added to the dialog
                '& .MuiDialog-container::after': {
                    content: 'none',
                },
            }}
        >
            {children}
        </Dialog>
    )
}