import { Dialog, useTheme } from "@mui/material";
import { UpTransition } from "../transitions";
import { LargeDialogProps } from "../types";

export const LargeDialog = ({
    children,
    id,
    isOpen,
    onClose,
    titleId,
    zIndex,
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
                zIndex: 1300,
                '& > .MuiDialog-container': {
                    '& > .MuiPaper-root': {
                        zIndex,
                        margin: { xs: 0, sm: 2, md: 4 },
                        minWidth: { xs: '100vw', sm: 'unset' },
                        maxWidth: { xs: '100vw', sm: 'calc(100vw - 64px)' },
                        bottom: { xs: 0, sm: 'auto' },
                        top: { xs: 'auto', sm: undefined },
                        position: { xs: 'absolute', sm: 'relative' },
                        display: { xs: 'block', sm: 'inline-block' },
                        background: palette.background.default,
                        color: palette.background.textPrimary,
                        '& > .MuiDialogContent-root': {
                            position: 'relative',
                        },
                    },
                }
            }}
        >
            {children}
        </Dialog>
    )
}