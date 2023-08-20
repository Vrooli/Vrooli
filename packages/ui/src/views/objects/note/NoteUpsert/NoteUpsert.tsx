import { endpointGetNoteVersion, endpointPostNoteVersion, endpointPutNoteVersion, NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteForm, noteInitialValues, transformNoteValues, validateNoteValues } from "forms/NoteForm/NoteForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { NoteUpsertProps } from "../types";

export const NoteUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: NoteUpsertProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<NoteVersion, NoteVersionShape>({
        ...endpointGetNoteVersion,
        objectType: "NoteVersion",
        overrideObject,
        transform: (data) => noteInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput>({
        display,
        endpointCreate: endpointPostNoteVersion,
        endpointUpdate: endpointPutNoteVersion,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="note-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<NoteVersionCreateInput | NoteVersionUpdateInput, NoteVersion>({
                        fetch,
                        inputs: transformNoteValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateNoteValues(values, existing, isCreate)}
            >
                {(formik) =>
                    <NoteForm
                        display={display}
                        isCreate={isCreate}
                        isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                        isOpen={true}
                        onCancel={handleCancel}
                        ref={formRef}
                        versions={[]}
                        zIndex={zIndex}
                        {...formik}
                    />
                }
            </Formik>
        </MaybeLargeDialog>
    );
};
