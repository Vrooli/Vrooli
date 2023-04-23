import { Box, List, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { ListContainerProps } from "../types";

export const ListContainer = ({
    children,
    emptyText,
    isEmpty = false,
    sx,
}: ListContainerProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    return (
        <Box sx={{
            marginTop: 2,
            maxWidth: "1000px",
            marginLeft: "auto",
            marginRight: "auto",
            ...(isEmpty ? {} : {
                boxShadow: 12,
                background: palette.background.paper,
                borderRadius: "8px",
                overflow: "overlay",
                display: "block",
            }),
            ...(sx ?? {}),
        }}>
            {isEmpty && (
                <Typography variant="h6" textAlign="center">
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
};
