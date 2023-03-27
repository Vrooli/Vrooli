import { FindVersionInput, RoutineVersion, RoutineVersionUpdateInput } from "@shared/consts";
import { routineVersionValidation } from '@shared/validation';
import { routineVersionFindOne } from "api/generated/endpoints/routineVersion_findOne";
import { routineVersionUpdate } from "api/generated/endpoints/routineVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from 'formik';
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineForm } from "forms/RoutineForm/RoutineForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { routineInitialValues } from "..";
import { RoutineUpdateProps } from "../types";

export const RoutineUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: RoutineUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<RoutineVersion, FindVersionInput>(routineVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<RoutineVersion>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<RoutineVersion, RoutineVersionUpdateInput>(routineVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateRoutine',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<RoutineVersion, RoutineVersionUpdateInput>({
                        mutation,
                        input: shapeRoutineVersion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={routineVersionValidation.update({})}
            >
                {(formik) => <RoutineForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
                    isOpen={true}
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