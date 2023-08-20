import { endpointGetStandardVersion, endpointPostStandardVersion, endpointPutStandardVersion, StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardForm, standardInitialValues, transformStandardValues, validateStandardValues } from "forms/StandardForm/StandardForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { StandardUpsertProps } from "../types";

export const StandardUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: StandardUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<StandardVersion, StandardVersionShape>({
        ...endpointGetStandardVersion,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => standardInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput>({
        display,
        endpointCreate: endpointPostStandardVersion,
        endpointUpdate: endpointPutStandardVersion,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="standard-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateStandard" : "UpdateStandard")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
                        fetch,
                        inputs: transformStandardValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateStandardValues(values, existing, isCreate)}
            >
                {(formik) => <StandardForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
