import { Status } from "@local/shared";
import { Stack, Tooltip, Typography } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { useCallback, useMemo } from "react";
import { usePopover } from "../../../hooks/usePopover.js";
import { RoutineIncompleteIcon, RoutineInvalidIcon, RoutineValidIcon } from "../../../icons/common.js";
import { noSelect } from "../../../styles.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { StatusButtonProps } from "../types.js";

/**
 * Status indicator and slider change color to represent routine's status
 */
const STATUS_COLOR = {
    [Status.Incomplete]: "#a0b121", // Yellow
    [Status.Invalid]: "#ff6a6a", // Red
    [Status.Valid]: "#00d51e", // Green
};
const STATUS_LABEL = {
    [Status.Incomplete]: "Incomplete",
    [Status.Invalid]: "Invalid",
    [Status.Valid]: "Valid",
};
const STATUS_ICON = {
    [Status.Incomplete]: RoutineIncompleteIcon,
    [Status.Invalid]: RoutineInvalidIcon,
    [Status.Valid]: RoutineValidIcon,
};

/**
 * Converts status messages to markdown display
 * @param messages List of status message strings 
 * @returns Indicator for no errors detected if no messages, just the error 
 * if one message, or a list of errors if multiple
 */
export function formatStatusMessages(messages: string[]) {
    if (!Array.isArray(messages) || messages.length === 0) return "No errors detected.";
    if (messages.length === 1) return messages[0];
    return messages.map(message => `* ${message}`).join("\n");
}

/**
 * Shows valid/invalid/incomplete status of some object
 */
export function StatusButton({
    status,
    messages,
    sx,
}: StatusButtonProps) {

    const statusMarkdown = useMemo(() => formatStatusMessages(messages), [messages]);

    const StatusIcon = useMemo(() => STATUS_ICON[status], [status]);

    const [anchorEl, open, close] = usePopover();
    const openPopover = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (!anchorEl) open(event);
    }, [anchorEl, open]);

    return (
        <>
            <Tooltip title='Press for details'>
                <Stack
                    direction="row"
                    spacing={1}
                    onClick={openPopover}
                    sx={{
                        ...noSelect,
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        cursor: "pointer",
                        background: STATUS_COLOR[status],
                        color: "white",
                        padding: "4px",
                        borderRadius: "16px",
                        ...(sx ?? {}),
                    }}>
                    <StatusIcon fill='white' />
                    <Typography
                        variant='body2'
                        sx={{
                            // Hide on small screens
                            display: { xs: "none", sm: "inline" },
                            paddingRight: "4px",
                        }}>{STATUS_LABEL[status]}</Typography>
                </Stack>
            </Tooltip>
            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={close}
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
                <MarkdownDisplay content={statusMarkdown} />
            </PopoverWithArrow>
        </>
    );
}
