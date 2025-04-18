import { endpointsNotification, FindByIdInput, Success } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ListItemChip, ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { NotificationListItemProps } from "../types.js";

export function NotificationListItem({
    data,
    onAction,
    ...props
}: NotificationListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [markAsReadMutation, { errors: markErrors }] = useLazyFetch<FindByIdInput, Success>(endpointsNotification.markAsRead);
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
                            <IconCommon name="Visible" fill="secondary.main" />
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
