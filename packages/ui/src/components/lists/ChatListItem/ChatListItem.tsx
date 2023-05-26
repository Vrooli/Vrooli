import { DeleteIcon } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ChatListItemProps } from "../types";

export function ChatListItem({
    data,
    onAction,
    ...props
}: ChatListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    return (
        <ObjectListItemBase
            {...props}
            toTheRight={
                <Stack direction="row" spacing={1}>
                    {/* {!data?.isRead && <Tooltip title={t("MarkAsRead")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("MarkAsRead", data.id)}>
                            <VisibleIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>} */}
                    <Tooltip title={t("Delete")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("Delete", data.id)}>
                            <DeleteIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            }
            data={data}
            objectType="Chat"
        />
    );
}
