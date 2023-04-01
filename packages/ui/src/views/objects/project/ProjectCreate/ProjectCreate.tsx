import { ProjectVersion, ProjectVersionCreateInput } from "@shared/consts";
import { projectVersionCreate } from "api/generated/endpoints/projectVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ProjectForm, projectInitialValues, validateProjectValues } from "forms/ProjectForm/ProjectForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeProjectVersion } from "utils/shape/models/projectVersion";
import { ProjectCreateProps } from "../types";

export const ProjectCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: ProjectCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => projectInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<ProjectVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<ProjectVersion, ProjectVersionCreateInput>(projectVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateProject',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<ProjectVersion, ProjectVersionCreateInput>({
                        mutation,
                        input: shapeProjectVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateProjectValues(values, true)}
            >
                {(formik) => <ProjectForm
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