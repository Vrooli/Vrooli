import { endpointGetSmartContractVersion, endpointPostSmartContractVersion, endpointPutSmartContractVersion, FindVersionInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { mutationWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractForm, smartContractInitialValues, transformSmartContractValues, validateSmartContractValues } from "forms/SmartContractForm/SmartContractForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SmartContractUpsertProps } from "../types";

export const SmartContractUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: SmartContractUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindVersionInput, SmartContractVersion>(endpointGetSmartContractVersion);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => smartContractInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<SmartContractVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<SmartContractVersionCreateInput, SmartContractVersion>(endpointPostSmartContractVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<SmartContractVersionUpdateInput, SmartContractVersion>(endpointPutSmartContractVersion);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateSmartContract" : "UpdateSmartContract",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<SmartContractVersion, SmartContractVersionUpdateInput>({
                        mutation,
                        input: transformSmartContractValues(values, existing),
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
