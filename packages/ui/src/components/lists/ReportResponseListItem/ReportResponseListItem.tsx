import { Box, useTheme } from "@mui/material";
import { ListItemChip, ObjectListItemBase } from "components/lists/ObjectListItemBase/ObjectListItemBase";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ReportResponseListItemProps } from "../types";

export function ReportResponseListItem({
    data,
    onAction,
    ...props
}: ReportResponseListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { title } = useMemo(() => {
        return {
            title: "",// data?.actionSuggested ? t(data.actionSuggested) : t("ReportResponse"),
        };
    }, [data, t]);

    return (
        <ObjectListItemBase
            {...props}
            data={data}
            objectType="ReportResponse"
            titleOverride={title}
            belowTags={
                <Box display="flex" flexDirection="row" alignItems="center" gap={1}>
                    {data?.created_at && (
                        <DateDisplay
                            showIcon={true}
                            timestamp={data.created_at}
                        />
                    )}
                    {data?.actionSuggested && (
                        <ListItemChip
                            color="Purple"
                            // label={t(data.actionSuggested)}
                            label={data.actionSuggested} //TODO
                            variant="outlined"
                            size="small"
                            sx={{
                                backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                                color: "white",
                            }}
                        />
                    )}
                </Box>
            }
            onAction={onAction}
        />
    );
}
