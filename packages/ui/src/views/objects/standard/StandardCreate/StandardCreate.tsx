import { StandardVersion, StandardVersionCreateInput } from "@shared/consts";
import { standardVersionCreate } from "api/generated/endpoints/standardVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardForm } from "forms/StandardForm/StandardForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeStandardVersion } from "utils/shape/models/standardVersion";
import { standardInitialValues, validateStandardValues } from "..";
import { StandardCreateProps } from "../types";

export const StandardCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: StandardCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => standardInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<StandardVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<StandardVersion, StandardVersionCreateInput>(standardVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateStandard',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<StandardVersion, StandardVersionCreateInput>({
                        mutation,
                        input: shapeStandardVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={async (values) => await validateStandardValues(values, true)}
            >
                {(formik) => <StandardForm
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