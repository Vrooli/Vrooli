import { endpointPutNotification, FindByIdInput, Success } from "@local/shared";
import { Chip, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { useLazyFetch } from "hooks/useLazyFetch";
import { VisibleIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { NotificationListItemProps } from "../types";

export function NotificationListItem({
    data,
    onAction,
    ...props
}: NotificationListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [markAsReadMutation, { errors: markErrors }] = useLazyFetch<FindByIdInput, Success>(endpointPutNotification);
    const onMarkAsRead = useCallback(() => {
        if (!data) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<FindByIdInput, Success>({
            fetch: markAsReadMutation,
            inputs: { id: data.id },
            successCondition: (data) => data.success,
            onSuccess: () => {
                onAction("Deleted", data.id);
            },
        });
    }, [data, markAsReadMutation, onAction]);

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
                        <IconButton edge="end" size="small" onClick={onMarkAsRead}>
                            <VisibleIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>}
                </Stack>
            }
            data={data}
            objectType="Notification"
            onAction={onAction}
        />
    );
}
