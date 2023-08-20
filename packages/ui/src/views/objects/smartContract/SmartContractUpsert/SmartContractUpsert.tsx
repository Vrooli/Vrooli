import { endpointGetSmartContractVersion, endpointPostSmartContractVersion, endpointPutSmartContractVersion, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractForm, smartContractInitialValues, transformSmartContractValues, validateSmartContractValues } from "forms/SmartContractForm/SmartContractForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { SmartContractShape } from "utils/shape/models/smartContract";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { SmartContractUpsertProps } from "../types";

export const SmartContractUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: SmartContractUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<SmartContractVersion, SmartContractVersionShape>({
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
        <MaybeLargeDialog
            display={display}
            id="smart-contract-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateSmartContract" : "UpdateSmartContract")}
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
                        inputs: transformSmartContractValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateSmartContractValues(values, existing, isCreate)}
            >
                {(formik) => <SmartContractForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={(existing?.root as SmartContractShape)?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
