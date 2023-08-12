import { endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineForm, routineInitialValues, transformRoutineValues, validateRoutineValues } from "forms/RoutineForm/RoutineForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RoutineShape } from "utils/shape/models/routine";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { RoutineUpsertProps } from "../types";

export const RoutineUpsert = ({
    isCreate,
    isOpen,
    isSubroutine = false,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: RoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput>({
        display,
        endpointCreate: endpointPostRoutineVersion,
        endpointUpdate: endpointPutRoutineVersion,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="routine-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
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
                        inputs: transformRoutineValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRoutineValues(values, existing, isCreate)}
            >
                {(formik) => <RoutineForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    isSubroutine={isSubroutine}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={(existing?.root as RoutineShape)?.versions?.map(v => v.versionLabel) ?? []}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
