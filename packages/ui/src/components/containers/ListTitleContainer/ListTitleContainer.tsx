/**
 * Title container with List inside
 */
// Used to display popular/search results of a particular object type
import { List, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TitleContainer } from "../TitleContainer/TitleContainer";
import { ListTitleContainerProps } from "../types";

export function ListTitleContainer({
    children,
    emptyText,
    isEmpty,
    ...props
}: ListTitleContainerProps) {
    const { t } = useTranslation();

    return (
        <TitleContainer {...props}>
            {
                isEmpty ?
                    <Typography variant="h6" pt={2} sx={{
                        textAlign: "center",
                    }}>{emptyText ?? t("NoResults", { ns: "error" })}</Typography> :
                    <List sx={{ overflow: "hidden" }}>
                        {children}
                    </List>
            }
        </TitleContainer>
    );
}
