import { CloseIcon } from "@local/shared";
import { Box, DialogTitle as MuiDialogTitle, IconButton, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { forwardRef } from "react";
import { noSelect } from "styles";
import { DialogTitleProps } from "../types";

export const DialogTitle = forwardRef(({
    below,
    helpText,
    id,
    onClose,
    title,
}: DialogTitleProps, ref) => {
    const { palette } = useTheme();

    return (
        <Box ref={ref} sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
        }}>
            <MuiDialogTitle
                id={id}
                sx={{
                    ...noSelect,
                    display: "flex",
                    alignItems: "center",
                    padding: 2,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                }}
            >
                <Box sx={{ marginLeft: "auto" }} >{title}</Box>
                {helpText && <HelpButton
                    markdown={helpText}
                    sx={{
                        fill: palette.secondary.light,
                    }}
                    sxRoot={{
                        display: "flex",
                        marginTop: "auto",
                        marginBottom: "auto",
                    }}
                />}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={onClose}
                    sx={{ marginLeft: "auto" }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </MuiDialogTitle>
            {below}
        </Box>
    );
});
