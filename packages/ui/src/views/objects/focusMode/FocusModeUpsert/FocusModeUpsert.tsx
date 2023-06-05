import { endpointGetFocusMode, endpointPostFocusMode, endpointPutFocusMode, FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeUpdateInput } from "@local/shared";
import { useLazyFetch } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { FocusModeForm, focusModeInitialValues, transformFocusModeValues, validateFocusModeValues } from "forms/FocusModeForm/FocusModeForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { FocusModeUpsertProps } from "../types";

export const FocusModeUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: FocusModeUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, FocusMode>(endpointGetFocusMode);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => focusModeInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<FocusMode>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<FocusModeCreateInput, FocusMode>(endpointPostFocusMode);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<FocusModeUpdateInput, FocusMode>(endpointPutFocusMode);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "FocusModeAdd" : "FocusModeUpdate",
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
                    mutationWrapper<FocusMode, FocusModeCreateInput | FocusModeUpdateInput>({
                        mutation,
                        input: transformFocusModeValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateFocusModeValues(values, existing)}
            >
                {(formik) => <FocusModeForm
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
