import { CloseIcon, useLocation } from "@local/shared";
import { Box, DialogTitle as MuiDialogTitle, IconButton, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { forwardRef } from "react";
import { noSelect } from "styles";
import { tryOnClose } from "utils/navigation/urlTools";
import { DialogTitleProps } from "../types";

export const DialogTitle = forwardRef(({
    below,
    id,
    onClose,
    ...titleData
}: DialogTitleProps, ref) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    return (
        <Box ref={ref} sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            ...titleData.sxs?.root,
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
                    variant="header"
                    {...titleData}
                    sxs={{
                        stack: { marginLeft: "auto", padding: 0 },
                    }}
                />
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={() => { tryOnClose(onClose, setLocation); }}
                    sx={{ marginLeft: "auto" }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </MuiDialogTitle>
            {below}
        </Box>
    );
});
