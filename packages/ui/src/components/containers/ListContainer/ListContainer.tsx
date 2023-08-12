import { List, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { BaseSection } from "styles";
import { ListContainerProps } from "../types";

export const ListContainer = ({
    children,
    emptyText,
    isEmpty = false,
    sx,
}: ListContainerProps) => {
    const { t } = useTranslation();

    return (
        <BaseSection sx={{
            maxWidth: "1000px",
            marginLeft: "auto",
            marginRight: "auto",
            padding: 0,
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
        </BaseSection>
    );
};
