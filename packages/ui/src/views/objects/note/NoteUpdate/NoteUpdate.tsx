import { FindVersionInput, NoteVersion, NoteVersionUpdateInput } from "@shared/consts";
import { noteVersionValidation } from '@shared/validation';
import { noteVersionFindOne } from "api/generated/endpoints/noteVersion_findOne";
import { noteVersionUpdate } from "api/generated/endpoints/noteVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteForm } from "forms/NoteForm/NoteForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeNoteVersion } from "utils/shape/models/noteVersion";
import { noteInitialValues } from "..";
import { NoteUpdateProps } from "../types";

export const NoteUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: NoteUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<NoteVersion, FindVersionInput>(noteVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => noteInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<NoteVersion>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<NoteVersion, NoteVersionUpdateInput>(noteVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateNote',
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
                    mutationWrapper<NoteVersion, NoteVersionUpdateInput>({
                        mutation,
                        input: shapeNoteVersion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={noteVersionValidation.update({})}
            >
                {(formik) => <NoteForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}