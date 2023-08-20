import { endpointGetFocusMode, endpointPostFocusMode, endpointPutFocusMode, FocusMode, FocusModeCreateInput, FocusModeUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { FocusModeForm, focusModeInitialValues, transformFocusModeValues, validateFocusModeValues } from "forms/FocusModeForm/FocusModeForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { FocusModeShape } from "utils/shape/models/focusMode";
import { FocusModeUpsertProps } from "../types";

export const FocusModeUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: FocusModeUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<FocusMode, FocusModeShape>({
        ...endpointGetFocusMode,
        objectType: "FocusMode",
        overrideObject,
        transform: (data) => focusModeInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<FocusMode, FocusModeCreateInput, FocusModeUpdateInput>({
        display,
        endpointCreate: endpointPostFocusMode,
        endpointUpdate: endpointPutFocusMode,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="focus-mode-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateFocusMode" : "UpdateFocusMode")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<FocusModeCreateInput | FocusModeUpdateInput, FocusMode>({
                        fetch,
                        inputs: transformFocusModeValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateFocusModeValues(values, existing, isCreate)}
            >
                {(formik) => <FocusModeForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
