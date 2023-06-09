import { CloseIcon } from "@local/shared";
import { Box, DialogTitle as MuiDialogTitle, IconButton, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { forwardRef } from "react";
import { noSelect } from "styles";
import { DialogTitleProps } from "../types";

export const DialogTitle = forwardRef(({
    below,
    id,
    onClose,
    ...titleData
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
                <Title
                    {...titleData}
                    sxs={{ stack: { marginLeft: "auto" } }}
                    variant="header"
                />
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
