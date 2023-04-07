import { FindVersionInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput } from "@shared/consts";
import { routineVersionCreate } from "api/generated/endpoints/routineVersion_create";
import { routineVersionFindOne } from "api/generated/endpoints/routineVersion_findOne";
import { routineVersionUpdate } from "api/generated/endpoints/routineVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from 'formik';
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineForm, routineInitialValues, transformRoutineValues, validateRoutineValues } from "forms/RoutineForm/RoutineForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RoutineUpsertProps } from "../types";

export const RoutineUpsert = ({
    display = 'page',
    isCreate,
    isSubroutine = false,
    onCancel,
    onCompleted,
    zIndex = 200,
}: RoutineUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<RoutineVersion, FindVersionInput>(routineVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<RoutineVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<RoutineVersion, RoutineVersionCreateInput>(routineVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<RoutineVersion, RoutineVersionUpdateInput>(routineVersionUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? 'CreateRoutine' : 'UpdateRoutine',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<RoutineVersion, RoutineVersionUpdateInput>({
                        mutation,
                        input: transformRoutineValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateRoutineValues(values, existing)}
            >
                {(formik) => <RoutineForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    isSubroutine={isSubroutine}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}