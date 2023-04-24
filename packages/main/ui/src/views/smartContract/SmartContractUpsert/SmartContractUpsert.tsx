import { FindVersionInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { mutationWrapper } from "../../../api";
import { smartContractVersionFindOne } from "../../../api/generated/endpoints/smartContractVersion_findOne";
import { smartContractCreate } from "../../../api/generated/endpoints/smartContract_create";
import { smartContractUpdate } from "../../../api/generated/endpoints/smartContract_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { BaseFormRef } from "../../../forms/BaseForm/BaseForm";
import { SmartContractForm, smartContractInitialValues, transformSmartContractValues, validateSmartContractValues } from "../../../forms/SmartContractForm/SmartContractForm";
import { SessionContext } from "../../../utils/SessionContext";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
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
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<SmartContractVersion, FindVersionInput>(smartContractVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => smartContractInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<SmartContractVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<SmartContractVersion, SmartContractVersionCreateInput>(smartContractCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<SmartContractVersion, SmartContractVersionUpdateInput>(smartContractUpdate);
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
