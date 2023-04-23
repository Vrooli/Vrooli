import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { DeleteIcon } from "@local/icons";
import { Box, Button } from "@mui/material";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { mutationWrapper } from "../../../api";
import { reminderCreate } from "../../../api/generated/endpoints/reminder_create";
import { reminderFindOne } from "../../../api/generated/endpoints/reminder_findOne";
import { reminderUpdate } from "../../../api/generated/endpoints/reminder_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "../../../forms/ReminderForm.tsx/ReminderForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const ReminderUpsert = ({ display = "page", handleDelete, isCreate, listId, onCancel, onCompleted, partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(reminderFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => reminderInitialValues(session, listId, { ...existing, ...partialData }), [existing, listId, partialData, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(reminderCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(reminderUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateReminder" : "UpdateReminder",
                }, below: !isCreate ? (_jsx(Box, { pb: 2, sx: { display: "flex", justifyContent: "center" }, children: _jsx(Button, { onClick: handleDelete, startIcon: _jsx(DeleteIcon, {}), children: t("Delete") }) })) : undefined }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformReminderValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateReminderValues(values, existing), children: (formik) => _jsx(ReminderForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=ReminderUpsert.js.map