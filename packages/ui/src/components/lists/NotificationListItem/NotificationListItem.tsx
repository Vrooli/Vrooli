import { Chip, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { DeleteIcon, VisibleIcon } from "icons";
import { useTranslation } from "react-i18next";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { NotificationListItemProps } from "../types";

export function NotificationListItem({
    data,
    onAction,
    ...props
}: NotificationListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.category && (
                    <Chip
                        label={data.category}
                        variant="outlined"
                        size="small"
                        sx={{
                            backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                            color: "white",
                            width: "fit-content",
                        }}
                    />
                )
            }
            toTheRight={
                <Stack direction="row" spacing={1}>
                    {!data?.isRead && <Tooltip title={t("MarkAsRead")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("MarkAsRead", data.id)}>
                            <VisibleIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>}
                    <Tooltip title={t("Delete")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("Delete", data.id)}>
                            <DeleteIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            }
            data={data}
            objectType="Notification"
        />
    );
}
