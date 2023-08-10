import { endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineForm, routineInitialValues, transformRoutineValues, validateRoutineValues } from "forms/RoutineForm/RoutineForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { RoutineUpsertProps } from "../types";

export const RoutineUpsert = ({
    display = "page",
    isCreate,
    isSubroutine = false,
    onCancel,
    onCompleted,
    zIndex,
}: RoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        objectType: "RoutineVersion",
        upsertTransform: (existing) => routineInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<RoutineVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<RoutineVersionCreateInput, RoutineVersion>(endpointPostRoutineVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<RoutineVersionUpdateInput, RoutineVersion>(endpointPutRoutineVersion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateRoutine" : "UpdateRoutine")}
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
                    fetchLazyWrapper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
                        fetch,
                        inputs: transformRoutineValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
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
    );
};
