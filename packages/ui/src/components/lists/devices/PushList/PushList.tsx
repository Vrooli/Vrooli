import { DeleteOneInput, DeleteType, PushDevice, PushDeviceTestInput, PushDeviceUpdateInput, Success, endpointPostDeleteOne, endpointPutPushDevice, endpointPutPushDeviceTest } from "@local/shared";
import { Box, Button, IconButton, ListItem, ListItemText, Stack, Tooltip, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, DeleteIcon, SendIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { setupPush } from "utils/push";
import { updateArray } from "utils/shape/general";
import { PushListItemProps, PushListProps } from "../types";
import { PubSub } from "utils/pubsub";

const InformationalColumn = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(1),
    paddingLeft: theme.spacing(2),
    marginRight: "auto",
}));

const listItemStyle = {
    display: "flex",
    padding: 1,
} as const;
const listItemTextStyle = { ...multiLineEllipsis(1) } as const;

export function PushListItem({
    handleDelete,
    handleUpdate,
    handleTestPush,
    index,
    data,
}: PushListItemProps) {
    const { palette } = useTheme();

    const onTestPush = useCallback(function onTestPushCallback() {
        handleTestPush(data);
    }, [data, handleTestPush]);

    const onDelete = useCallback(function onDeleteCallback() {
        handleDelete(data);
    }, [data, handleDelete]);

    return (
        <ListItem
            disablePadding
            sx={listItemStyle}
        >
            <InformationalColumn>
                <ListItemText
                    primary={data.name ?? data.id}
                    secondary={data.deviceId}
                    sx={listItemTextStyle}
                />
            </InformationalColumn>
            <Stack direction="row" spacing={1}>
                <Tooltip title={"Send test notification"}>
                    <IconButton onClick={onTestPush}>
                        <SendIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={"Delete push device"}>
                    <IconButton onClick={onDelete}>
                        <DeleteIcon fill={palette.error.main} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}

let isSettingUpPush = false;

const addButtonStyle = {
    marginTop: 2,
    marginBottom: 6,
} as const;

//TODO need way to name push devices
/**
 * Displays a list of push devices for the user to manage
 */
export function PushList({
    handleUpdate,
    list,
}: PushListProps) {
    const { t } = useTranslation();

    const [loadingAdd, setLoadingAdd] = useState<boolean>(isSettingUpPush);
    const handleAdd = useCallback(async function handleAddCallback() {
        if (isSettingUpPush) return;
        isSettingUpPush = true;
        setLoadingAdd(true);
        const createdPushObject = await setupPush(true);
        isSettingUpPush = false;
        setLoadingAdd(false);
        if (createdPushObject) {
            handleUpdate([...list, createdPushObject]);
        }
    }, [handleUpdate, list]);

    const [updateMutation, { loading: loadingUpdate }] = useLazyFetch<PushDeviceUpdateInput, PushDevice>(endpointPutPushDevice);
    const onUpdate = useCallback((index: number, updatedDevice: PushDevice) => {
        if (loadingUpdate) return;
        fetchLazyWrapper<PushDeviceUpdateInput, PushDevice>({
            fetch: updateMutation,
            inputs: {
                id: updatedDevice.id,
                name: updatedDevice.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedDevice));
            },
        });
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const onDelete = useCallback((device: PushDevice) => {
        if (loadingDelete) return;
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id: device.id, objectType: DeleteType.Email },
            onSuccess: () => {
                handleUpdate([...list.filter(w => w.id !== device.id)]);
            },
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete]);

    const [testPushNotification, { loading: loadingTestPush }] = useLazyFetch<PushDeviceTestInput, Success>(endpointPutPushDeviceTest);
    const onTestPush = useCallback((device: PushDevice) => {
        if (loadingTestPush) return;
        fetchLazyWrapper<PushDeviceTestInput, Success>({
            fetch: testPushNotification,
            inputs: { id: device.id },
            onSuccess: () => {
                PubSub.get().publish("snack", { message: "Test push sent", severity: "Success" });
            },
        });
    }, [loadingTestPush, testPushNotification]);

    return (
        <Box>
            <ListContainer
                emptyText={t("NoPushDevices", { ns: "error" })}
                isEmpty={list.length === 0}
            >
                {/* Push device list */}
                {list.map((device: PushDevice, index) => (
                    <PushListItem
                        key={`push-${index}`}
                        data={device}
                        index={index}
                        handleUpdate={onUpdate}
                        handleTestPush={onTestPush}
                        handleDelete={onDelete}
                    />
                ))}
            </ListContainer>
            <Button
                disabled={loadingAdd}
                fullWidth
                onClick={handleAdd}
                startIcon={<AddIcon />}
                variant="outlined"
                sx={addButtonStyle}
            >{t("AddThisDevice")}</Button>
        </Box>
    );
}
