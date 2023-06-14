import { EditIcon, ErrorIcon } from "@local/shared";
import CircularProgress from "@mui/material/CircularProgress";
import { green, red } from "@mui/material/colors";
import IconButton from "@mui/material/IconButton";
import { ChatBubbleStatusProps } from "components/types";
import { useEffect, useState } from "react";

/**
 * Provides a visual indicator for the status of a chat message.
 * It shows a CircularProgress that progresses as the message is sending,
 * and changes color and icon based on the success or failure of the operation.
 */
export const ChatBubbleStatus = ({
    hasError,
    isEditing,
    isSending,
    onEdit,
    onRetry,
}: ChatBubbleStatusProps) => {
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);

    useEffect(() => {
        // Updates the progress value every 100ms, but stops at 90 if the message is still sending
        let timer: NodeJS.Timeout;
        if (isSending) {
            timer = setInterval(() => {
                setProgress((oldProgress) => {
                    if (oldProgress === 100) {
                        setIsCompleted(true);
                        return 100;
                    }
                    const diff = 3;
                    return Math.min(oldProgress + diff, isSending ? 90 : 100);
                });
            }, 50);
        }

        // Cleans up the interval when the component is unmounted or the message is not sending anymore
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [isSending]);

    useEffect(() => {
        // Resets the progress and completion state after a delay when the sending has completed
        if (isCompleted && !isSending) {
            const timer = setTimeout(() => {
                setProgress(0);
                setIsCompleted(false);
            }, 1000);
            return () => {
                clearTimeout(timer);
            };
        }
    }, [isCompleted, isSending]);

    // While the message is sending or has just completed, show a CircularProgress
    if (isSending || isCompleted) {
        return (
            <CircularProgress
                variant="determinate"
                value={progress}
                size={24}
                sx={{ color: isSending ? "secondary.main" : hasError ? red[500] : green[500] }}
            />
        );
    }

    // If editing, don't show any icon
    if (isEditing) {
        return null;
    }
    // If there was an error, show an ErrorIcon
    if (hasError) {
        return (
            <IconButton onClick={() => { onRetry(); }} sx={{ color: red[500] }}>
                <ErrorIcon />
            </IconButton>
        );
    }
    // Otherwise, show an EditIcon
    return (
        <IconButton onClick={() => { onEdit(); }} sx={{ color: green[500] }}>
            <EditIcon />
        </IconButton>
    );
};
