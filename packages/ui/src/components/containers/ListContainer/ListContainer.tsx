import { Box, List, Typography, useTheme } from "@mui/material";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { ListContainerProps } from "../types";

export const ListContainer = forwardRef<HTMLDivElement, ListContainerProps>(({
    children,
    emptyText,
    id,
    isEmpty = false,
    sx,
}, ref) => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();

    return (
        <Box id={id} ref={ref} sx={{
            maxWidth: "1000px",
            width: "100%",
            marginLeft: "auto",
            marginRight: "auto",
            ...(isEmpty ? {} : {
                background: palette.background.paper,
                borderRadius: "8px",
                overflow: "overlay",
                display: "block",
            }),
            [breakpoints.down("sm")]: {
                borderRadius: 0,
            },
            ...(sx ?? {}),
        }}>
            {isEmpty && (
                <Typography variant="h6" textAlign="center" pt={2}>
                    {emptyText ?? t("NoResults", { ns: "error" })}
                </Typography>
            )}
            {!isEmpty && (
                <List sx={{ padding: 0 }}>
                    {children}
                </List>
            )}
        </Box>
    );
});
ListContainer.displayName = "ListContainer";
