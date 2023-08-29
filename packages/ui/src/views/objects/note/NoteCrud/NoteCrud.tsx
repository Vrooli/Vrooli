import { endpointGetNoteVersion, endpointPostNoteVersion, endpointPutNoteVersion, NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { NoteForm, noteInitialValues, transformNoteValues, validateNoteValues } from "forms/NoteForm/NoteForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useMemo } from "react";
import { getYou, ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { NoteCrudProps } from "../types";

export const NoteCrud = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    onDeleted,
    overrideObject,
}: NoteCrudProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<NoteVersion, NoteVersionShape>({
        ...endpointGetNoteVersion,
        objectType: "NoteVersion",
        overrideObject,
        transform: (data) => noteInitialValues(session, data),
    });
    const { canUpdate } = useMemo(() => getYou(existing as ListObject), [existing]);

    const {
        fetch,
        handleCancel,
        handleCompleted,
        handleDeleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput>({
        display,
        endpointCreate: endpointPostNoteVersion,
        endpointUpdate: endpointPutNoteVersion,
        isCreate,
        onCancel,
        onCompleted,
        onDeleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="note-crud-dialog"
            isOpen={isOpen ?? false}
            onClose={handleClose}
            sxs={{ paper: { height: "100%" } }}
        >
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!(isCreate || canUpdate)) {
                        PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
                        return;
                    }
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
                    <>
                        <NoteForm
                            disabled={!(isCreate || canUpdate)}
                            display={display}
                            handleClose={handleClose}
                            handleDeleted={handleDeleted}
                            isCreate={isCreate}
                            isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                            isOpen={true}
                            onCancel={handleCancel}
                            onClose={handleClose}
                            ref={formRef}
                            versions={[]}
                            {...formik}
                        />
                    </>
                }
            </Formik>
        </MaybeLargeDialog>
    );
};
