import { endpointGetProjectVersion, endpointPostProjectVersion, endpointPutProjectVersion, ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ProjectForm, projectInitialValues, transformProjectValues, validateProjectValues } from "forms/ProjectForm/ProjectForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { ProjectUpsertProps } from "../types";

export const ProjectUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: ProjectUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<ProjectVersion, ProjectVersionShape>({
        ...endpointGetProjectVersion,
        objectType: "ProjectVersion",
        upsertTransform: (existing) => projectInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<ProjectVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ProjectVersionCreateInput, ProjectVersion>(endpointPostProjectVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ProjectVersionUpdateInput, ProjectVersion>(endpointPutProjectVersion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ProjectVersionCreateInput | ProjectVersionUpdateInput, ProjectVersion>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateProject" : "UpdateProject")}
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
                    fetchLazyWrapper<ProjectVersionCreateInput | ProjectVersionUpdateInput, ProjectVersion>({
                        fetch,
                        inputs: transformProjectValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateProjectValues(values, existing)}
            >
                {(formik) => <ProjectForm
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
