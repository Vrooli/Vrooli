import { FindVersionInput, ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput } from "@shared/consts";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { projectVersionCreate } from "../../../api/generated/endpoints/projectVersion_create";
import { projectVersionFindOne } from "../../../api/generated/endpoints/projectVersion_findOne";
import { projectVersionUpdate } from "../../../api/generated/endpoints/projectVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { BaseFormRef } from "../../../forms/BaseForm/BaseForm";
import { ProjectForm, projectInitialValues, transformProjectValues, validateProjectValues } from "../../../forms/ProjectForm/ProjectForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { ProjectUpsertProps } from "../types";

export const ProjectUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: ProjectUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<ProjectVersion, FindVersionInput>(projectVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => projectInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<ProjectVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<ProjectVersion, ProjectVersionCreateInput>(projectVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<ProjectVersion, ProjectVersionUpdateInput>(projectVersionUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateProject" : "UpdateProject",
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
                    mutationWrapper<ProjectVersion, ProjectVersionUpdateInput>({
                        mutation,
                        input: transformProjectValues(values, existing),
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
