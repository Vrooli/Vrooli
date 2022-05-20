/**
 * A button that looks like a link
 */
import { Button, Typography, useTheme } from "@mui/material"
import { LinkButtonProps } from "../types";

export const LinkButton = ({
    onClick,
    text,
    sxs,
}: LinkButtonProps) => {
    const { palette } = useTheme();

    return (
        <Button
            variant="text"
            onClick={onClick}
            sx={{
                minWidth: 'auto',
                padding: 0,
                // Don't transform text
                textTransform: 'none',
                // Highlight text like a link
                textDecoration: 'none',
                '&:hover': {
                    textDecoration: 'underline',
                },
                ...sxs?.button,
            }}
        >
            <Typography
                variant="body1"
                sx={{
                    color: palette.primary.contrastText,
                    cursor: 'pointer',
                    ...sxs?.text,
                }}
            >
                {text}
            </Typography>
        </Button>
    )
}