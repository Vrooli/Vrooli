import { RoutineVersion, RoutineVersionCreateInput } from "@shared/consts";
import { routineVersionCreate } from "api/generated/endpoints/routineVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineForm, routineInitialValues, validateRoutineValues } from "forms/RoutineForm/RoutineForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineCreateProps } from "../types";

export const RoutineCreate = ({
    display = 'page',
    isSubroutine = false,
    onCancel,
    onCreated,
    zIndex = 200,
}: RoutineCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => routineInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<RoutineVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<RoutineVersion, RoutineVersionCreateInput>(routineVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateRoutine',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<RoutineVersion, RoutineVersionCreateInput>({
                        mutation,
                        input: shapeRoutineVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateRoutineValues(values, true)}
            >
                {(formik) => <RoutineForm
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