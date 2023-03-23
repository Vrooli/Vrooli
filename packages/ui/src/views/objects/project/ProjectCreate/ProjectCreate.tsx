import { ProjectVersion, ProjectVersionCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { projectVersionValidation } from '@shared/validation';
import { projectVersionCreate } from "api/generated/endpoints/projectVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { userFromSession } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ProjectForm } from "forms/ProjectForm/ProjectForm";
import { useContext, useRef } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeProjectVersion } from "utils/shape/models/projectVersion";
import { ProjectCreateProps } from "../types";

export const ProjectCreate = ({
    display = 'page',
    zIndex = 200,
}: ProjectCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onCreated } = useCreateActions<ProjectVersion>();
    const [mutation, { loading: isLoading }] = useCustomMutation<ProjectVersion, ProjectVersionCreateInput>(projectVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateProject',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'ProjectVersion' as const,
                    id: uuid(),
                    isComplete: false,
                    isPrivate: false,
                    root: {
                        __typename: 'Project' as const,
                        id: uuid(),
                        owner: session ? userFromSession(session!) : null,
                        parent: null,
                        project: null,
                    },
                    translations: [{
                        id: uuid(),
                        language: getUserLanguages(session)[0],
                        name: '',
                        description: '',
                    }],
                    versionLabel: '1.0.0',
                    versionNotes: '',
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<ProjectVersion, ProjectVersionCreateInput>({
                        mutation,
                        input: shapeProjectVersion.create(values),
                        onSuccess: (data) => { onCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={projectVersionValidation.create({})}
            >
                {(formik) => <ProjectForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}