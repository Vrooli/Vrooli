import { endpointGetNoteVersion, endpointPostNoteVersion, endpointPutNoteVersion, FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteForm, noteInitialValues, transformNoteValues, validateNoteValues } from "forms/NoteForm/NoteForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { NoteUpsertProps } from "../types";

export const NoteUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: NoteUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl({}), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindVersionInput, NoteVersion>(endpointGetNoteVersion);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => noteInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<NoteVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<NoteVersionCreateInput, NoteVersion>(endpointPostNoteVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<NoteVersionUpdateInput, NoteVersion>(endpointPutNoteVersion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<NoteVersionCreateInput | NoteVersionUpdateInput, NoteVersion>;

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                if (!isCreate && !existing) {
                    PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                    return;
                }
                fetchLazyWrapper<NoteVersionCreateInput | NoteVersionUpdateInput, NoteVersion>({
                    fetch,
                    inputs: transformNoteValues(values, existing),
                    onSuccess: (data) => { handleCompleted(data); },
                    onError: () => { helpers.setSubmitting(false); },
                });
            }}
            validate={async (values) => await validateNoteValues(values, existing)}
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
    );
};
