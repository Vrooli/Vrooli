import { DeleteOneInput, DeleteType, endpointPostDeleteOne, endpointPostPushDevice, endpointPutPushDevice, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, pushDeviceValidation, Success } from "@local/shared";
import { Button, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { useFormik } from "formik";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { getDeviceInfo } from "utils/display/device";
import { PubSub } from "utils/pubsub";
import { setupPush } from "utils/push";
import { updateArray } from "utils/shape/general";
import { PushListItemProps, PushListProps } from "../types";

//  TODO copied from emaillistitem. need to rewrite
export function PushListItem({
    handleDelete,
    handleUpdate,
    index,
    data,
}: PushListItemProps) {
    const { palette } = useTheme();

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: "flex",
                padding: 1,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: "auto" }}>
                <ListItemText
                    primary={data.name ?? data.id}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                {/* <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                    color: data.verified ? Status.Verified : Status.NotVerified,
                    height: 'fit-content',
                    fontWeight: 'bold',
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    textAlign: 'center',
                    padding: 0.25,
                    width: 'fit-content',
                }}>
                    {data.verified ? "Verified" : "Not Verified"}
                </Box> */}
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                <Tooltip title="Delete Email">
                    <IconButton
                        onClick={onDelete}
                    >
                        <DeleteIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}

//TODO copied from emaillist. need to rewrite
/**
 * Displays a list of push devices for the user to manage
 */
export const PushList = ({
    handleUpdate,
    list,
}: PushListProps) => {
    const { t } = useTranslation();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useLazyFetch<PushDeviceCreateInput, PushDevice>(endpointPostPushDevice);
    const formik = useFormik({
        initialValues: {
            endpoint: "",
            expires: "",
            keys: {
                auth: "",
                p256dh: "",
            },
        },
        enableReinitialize: true,
        validationSchema: pushDeviceValidation.create({ env: process.env.NODE_ENV as "development" | "production" }),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            fetchLazyWrapper<PushDeviceCreateInput, PushDevice>({
                fetch: addMutation,
                inputs: {
                    endpoint: values.endpoint,
                    expires: values.expires,
                    keys: values.keys,
                    name: getDeviceInfo().deviceName,
                },
                onSuccess: (data) => {
                    PubSub.get().publish("snack", { messageKey: "CompleteVerificationInEmail", severity: "Info" });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });

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

    return (
        <form onSubmit={formik.handleSubmit}>
            <ListContainer
                emptyText={t("NoPushDevices", { ns: "error" })}
                isEmpty={list.length === 0}
                sx={{ maxWidth: "500px" }}
            >
                {/* Push device list */}
                {list.map((device: PushDevice, index) => (
                    <PushListItem
                        key={`push-${index}`}
                        data={device}
                        index={index}
                        handleUpdate={onUpdate}
                        handleDelete={onDelete}
                    />
                ))}
            </ListContainer>
            {/* Add new push-device */}
            <Stack direction="row" sx={{
                display: "flex",
                justifyContent: "center",
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                <Button
                    disabled={loadingAdd}
                    fullWidth
                    onClick={() => { setupPush(true); }}
                    startIcon={<AddIcon />}
                    variant="outlined"
                >{t("AddThisDevice")}</Button>
            </Stack>
        </form>
    );
};
