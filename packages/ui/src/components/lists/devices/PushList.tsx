import { DeleteOneInput, DeleteType, PushDevice, PushDeviceTestInput, PushDeviceUpdateInput, Success, endpointsActions, endpointsPushDevice, updateArray } from "@local/shared";
import { Box, Button, IconButton, ListItem, ListItemText, Stack, TextField, Tooltip, styled, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useEditableLabel } from "../../../hooks/useEditableLabel.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { multiLineEllipsis } from "../../../styles.js";
import { PubSub } from "../../../utils/pubsub.js";
import { setupPush } from "../../../utils/push.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { PushListItemProps, PushListProps } from "./types.js";

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

const textFieldStyle = {
    flexGrow: 1,
    width: "auto",
    "& .MuiInputBase-input": {
        padding: 0,
    },
} as const;

export function PushListItem({
    handleDelete,
    handleUpdate,
    handleTestPush,
    index,
    data,
}: PushListItemProps) {
    const { palette, typography } = useTheme();

    const onTestPush = useCallback(function onTestPushCallback() {
        handleTestPush(data);
    }, [data, handleTestPush]);

    const onDelete = useCallback(function onDeleteCallback() {
        handleDelete(data);
    }, [data, handleDelete]);

    const updateLabel = useCallback((updatedLabel: string) => {
        handleUpdate(index, { ...data, name: updatedLabel });
    }, [data, handleUpdate, index]);
    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        isEditingLabel,
        labelEditRef,
        startEditingLabel,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: true,
        isMultiline: false,
        label: data.name || data.deviceId || "",
        onUpdate: updateLabel,
    });

    const textFieldInputProps = useMemo(function textFieldInputPropsMemo() {
        return { style: (typography.subtitle1 as object || {}) };
    }, [typography]);
    const listItemTextStyle = useMemo(function listItemTextStyleMemo() {
        return {
            ...multiLineEllipsis(1),
            cursor: "pointer",
            // Ensure there's a clickable area, even if the text is empty
            minWidth: "20px",
            minHeight: "24px",
            "&:empty::before": {
                content: "\"\"",
                display: "inline-block",
                width: "100%",
                height: "100%",
            },
        };
    }, []);

    return (
        <ListItem
            disablePadding
            sx={listItemStyle}
        >
            <InformationalColumn>
                {isEditingLabel ? (
                    <TextField
                        ref={labelEditRef}
                        inputMode="text"
                        InputProps={textFieldInputProps}
                        onBlur={submitLabelChange}
                        onChange={handleLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        placeholder={"Enter name..."}
                        size="small"
                        value={editedLabel}
                        sx={textFieldStyle}
                    />) : (
                    <Box display="flex" flexDirection="row" gap={1} justifyContent="center" alignItems="center">
                        <ListItemText
                            primary={editedLabel}
                            sx={listItemTextStyle}
                            onClick={startEditingLabel}
                        />
                        <IconButton onClick={startEditingLabel}>
                            <IconCommon
                                decorative
                                fill="secondary.main"
                                name="Edit"
                            />
                        </IconButton>
                    </Box>
                )}
            </InformationalColumn>
            <Stack direction="row" spacing={1}>
                <Tooltip title={"Send test notification"}>
                    <IconButton onClick={onTestPush}>
                        <IconCommon
                            decorative
                            fill="secondary.main"
                            name="Send"
                        />
                    </IconButton>
                </Tooltip>
                <Tooltip title={"Delete push device"}>
                    <IconButton onClick={onDelete}>
                        <IconCommon
                            decorative
                            fill="error.main"
                            name="Delete"
                        />
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

    const [updateMutation, { loading: loadingUpdate }] = useLazyFetch<PushDeviceUpdateInput, PushDevice>(endpointsPushDevice.updateOne);
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

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
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

    const [testPushNotification, { loading: loadingTestPush }] = useLazyFetch<PushDeviceTestInput, Success>(endpointsPushDevice.testOne);
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
                startIcon={<IconCommon
                    decorative
                    name="Add"
                />}
                variant="outlined"
                sx={addButtonStyle}
            >{t("AddThisDevice")}</Button>
        </Box>
    );
}
