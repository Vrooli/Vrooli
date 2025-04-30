import { Box, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { DateDisplay } from "../../text/DateDisplay.js";
import { ListItemChip, ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { ReportResponseListItemProps } from "../types.js";

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
                    {data?.createdAt && (
                        <DateDisplay
                            showIcon={true}
                            timestamp={data.createdAt}
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
