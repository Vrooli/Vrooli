import { SmartContractVersion, SmartContractVersionCreateInput } from "@shared/consts";
import { mutationWrapper } from "api";
import { smartContractVersionCreate } from "api/generated/endpoints/smartContractVersion_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from 'formik';
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractForm } from "forms/SmartContractForm/SmartContractForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeSmartContractVersion } from "utils/shape/models/smartContractVersion";
import { smartContractInitialValues, validateSmartContractValues } from "..";
import { SmartContractCreateProps } from "../types";

export const SmartContractCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: SmartContractCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => smartContractInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<SmartContractVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<SmartContractVersion, SmartContractVersionCreateInput>(smartContractVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateSmartContract',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<SmartContractVersion, SmartContractVersionCreateInput>({
                        mutation,
                        input: shapeSmartContractVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateSmartContractValues(values, true)}
            >
                {(formik) => <SmartContractForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={[]}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}