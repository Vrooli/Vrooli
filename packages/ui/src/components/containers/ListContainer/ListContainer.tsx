import { Box, List, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { ListContainerProps } from "../types";

export const ListContainer = ({
    children,
    emptyText,
    isEmpty = false,
    sx,
}: ListContainerProps) => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();

    return (
        <Box sx={{
            maxWidth: "1000px",
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
