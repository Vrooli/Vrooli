import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@shared/consts";
import { noteVersionCreate } from "api/generated/endpoints/noteVersion_create";
import { noteVersionFindOne } from "api/generated/endpoints/noteVersion_findOne";
import { noteVersionUpdate } from "api/generated/endpoints/noteVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteForm, noteInitialValues, transformNoteValues, validateNoteValues } from "forms/NoteForm/NoteForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { NoteUpsertProps } from "../types";

export const NoteUpsert = ({
    display = 'page',
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: NoteUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<NoteVersion, FindVersionInput>(noteVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => noteInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<NoteVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<NoteVersion, NoteVersionCreateInput>(noteVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<NoteVersion, NoteVersionUpdateInput>(noteVersionUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? 'CreateNote' : 'UpdateNote',
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
                    mutationWrapper<NoteVersion, NoteVersionCreateInput | NoteVersionUpdateInput>({
                        mutation,
                        input: transformNoteValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateNoteValues(values, existing)}
            >
                {(formik) => <NoteForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={[]}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}