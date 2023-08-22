/**
 * Shows valid/invalid/incomplete status of some object
 */
import { Stack, Tooltip, Typography } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { RoutineIncompleteIcon, RoutineInvalidIcon, RoutineValidIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "styles";
import { Status } from "utils/consts";
import { MarkdownDisplay } from "../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { StatusButtonProps } from "../types";

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

export const StatusButton = ({
    status,
    messages,
    sx,
}: StatusButtonProps) => {
    /**
     * List of status messages converted to markdown. 
     * If one message, no bullet points. If multiple, bullet points.
     */
    const statusMarkdown = useMemo(() => {
        if (messages.length === 0) return "No errors detected.";
        if (messages.length === 1) {
            return messages[0];
        }
        return messages.map((s) => {
            return `* ${s}`;
        }).join("\n");
    }, [messages]);

    const StatusIcon = useMemo(() => STATUS_ICON[status], [status]);

    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const openPopover = useCallback((event: any) => {
        console.log("openPopover anchor", event.currentTarget);
        if (!anchorEl) setAnchorEl(event.currentTarget);
    }, [anchorEl]);
    const closePopover = () => {
        setAnchorEl(null);
    };

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
                <MarkdownDisplay content={statusMarkdown} sx={{ minHeight: "unset" }} />
            </PopoverWithArrow>
        </>
    );
};
