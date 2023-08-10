import { endpointGetSmartContractVersion, endpointPostSmartContractVersion, endpointPutSmartContractVersion, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractForm, smartContractInitialValues, transformSmartContractValues, validateSmartContractValues } from "forms/SmartContractForm/SmartContractForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { SmartContractUpsertProps } from "../types";

export const SmartContractUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: SmartContractUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<SmartContractVersion, SmartContractVersionShape>({
        ...endpointGetSmartContractVersion,
        objectType: "SmartContractVersion",
        overrideObject,
        transform: (existing) => smartContractInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput>({
        display,
        endpointCreate: endpointPostSmartContractVersion,
        endpointUpdate: endpointPutSmartContractVersion,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateSmartContract" : "UpdateSmartContract")}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<SmartContractVersionCreateInput | SmartContractVersionUpdateInput, SmartContractVersion>({
                        fetch,
                        inputs: transformSmartContractValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateSmartContractValues(values, existing)}
            >
                {(formik) => <SmartContractForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
