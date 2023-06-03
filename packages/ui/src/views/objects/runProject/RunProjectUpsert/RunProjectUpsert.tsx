import { FindByIdInput, RunProject, runProjectCreate, RunProjectCreateInput, runProjectUpdate, RunProjectUpdateInput } from "@local/shared";
import { useCustomMutation } from "api";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunProjectForm, runProjectInitialValues, transformRunProjectValues, validateRunProjectValues } from "forms/RunProjectForm/RunProjectForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RunProjectUpsertProps } from "../types";

export const RunProjectUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: RunProjectUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, RunProject>("/runProject");
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => runProjectInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<RunProject>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<RunProject, RunProjectCreateInput>(runProjectCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<RunProject, RunProjectUpdateInput>(runProjectUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateRun" : "UpdateRun",
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
                    mutationWrapper<RunProject, RunProjectCreateInput | RunProjectUpdateInput>({
                        mutation,
                        input: transformRunProjectValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRunProjectValues(values, existing)}
            >
                {(formik) => <RunProjectForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
