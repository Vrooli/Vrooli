import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { DeleteType } from "@local/consts";
import { AddIcon } from "@local/icons";
import { pushDeviceValidation } from "@local/validation";
import { Button, Stack, useTheme } from "@mui/material";
import { useFormik } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { deleteOneOrManyDeleteOne } from "../../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { pushDeviceCreate } from "../../../../../api/generated/endpoints/pushDevice_create";
import { pushDeviceUpdate } from "../../../../../api/generated/endpoints/pushDevice_update";
import { useCustomMutation } from "../../../../../api/hooks";
import { mutationWrapper } from "../../../../../api/utils";
import { getDeviceInfo } from "../../../../../utils/display/device";
import { PubSub } from "../../../../../utils/pubsub";
import { setupPush } from "../../../../../utils/push";
import { updateArray } from "../../../../../utils/shape/general";
import { ListContainer } from "../../../../containers/ListContainer/ListContainer";
import { PushListItem } from "../PushListItem/PushListItem";
export const PushList = ({ handleUpdate, list, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [addMutation, { loading: loadingAdd }] = useCustomMutation(pushDeviceCreate);
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
        validationSchema: pushDeviceValidation.create({}),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd)
                return;
            mutationWrapper({
                mutation: addMutation,
                input: {
                    endpoint: values.endpoint,
                    expires: values.expires,
                    keys: values.keys,
                    name: getDeviceInfo().deviceName,
                },
                onSuccess: (data) => {
                    PubSub.get().publishSnack({ messageKey: "CompleteVerificationInEmail", severity: "Info" });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });
    const [updateMutation, { loading: loadingUpdate }] = useCustomMutation(pushDeviceUpdate);
    const onUpdate = useCallback((index, updatedDevice) => {
        if (loadingUpdate)
            return;
        mutationWrapper({
            mutation: updateMutation,
            input: {
                id: updatedDevice.id,
                name: updatedDevice.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedDevice));
            },
        });
    }, [handleUpdate, list, loadingUpdate, updateMutation]);
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((device) => {
        if (loadingDelete)
            return;
        mutationWrapper({
            mutation: deleteMutation,
            input: { id: device.id, objectType: DeleteType.Email },
            onSuccess: () => {
                handleUpdate([...list.filter(w => w.id !== device.id)]);
            },
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete]);
    return (_jsxs("form", { onSubmit: formik.handleSubmit, children: [_jsx(ListContainer, { emptyText: t("NoPushDevices", { ns: "error" }), isEmpty: list.length === 0, sx: { maxWidth: "500px" }, children: list.map((device, index) => (_jsx(PushListItem, { data: device, index: index, handleUpdate: onUpdate, handleDelete: onDelete }, `push-${index}`))) }), _jsx(Stack, { direction: "row", sx: {
                    display: "flex",
                    justifyContent: "center",
                    paddingTop: 2,
                    paddingBottom: 6,
                }, children: _jsx(Button, { disabled: loadingAdd, fullWidth: true, onClick: setupPush, startIcon: _jsx(AddIcon, {}), children: t("AddThisDevice") }) })] }));
};
//# sourceMappingURL=PushList.js.map