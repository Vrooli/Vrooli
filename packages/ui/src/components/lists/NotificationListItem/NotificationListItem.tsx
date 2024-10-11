import { endpointPutNotification, FindByIdInput, Success } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { useLazyFetch } from "hooks/useLazyFetch";
import { VisibleIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { ListItemChip, ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
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
                    <ListItemChip
                        color="Purple"
                        label={data.category}
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
