import { Box, Dialog } from "@mui/material";
import { StatsCompact } from "components/text";
import { StatsCompactPropsObject } from "components/text/types"
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { StatsDialogProps } from "../types"

const titleAria = 'stats-object-dialog-title';

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsDialog = <T extends StatsCompactPropsObject>({
    handleObjectUpdate,
    isOpen,
    object,
    onClose,
    session,
    zIndex,
}: StatsDialogProps<T>) => {

    return (
        <Dialog
            onClose={onClose}
            open={isOpen}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    overflow: 'hidden',
                    borderRadius: 2,
                    boxShadow: 12,
                    textAlign: "center",
                    padding: "1em",
                },
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title="Share"
                onClose={onClose}
            />
            <Box sx={{ padding: 2 }}>
                {/* Bookmarks, votes, and other info */}
                <StatsCompact
                    handleObjectUpdate={handleObjectUpdate}
                    loading={false}
                    object={object}
                    session={session}
                />
                {/* Historical stats */}
                {/* TODO */}
            </Box>
        </Dialog>
    )
}