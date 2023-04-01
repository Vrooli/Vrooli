import { NoteVersion, NoteVersionCreateInput } from "@shared/consts";
import { noteVersionCreate } from "api/generated/endpoints/noteVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteForm, noteInitialValues, validateNoteValues } from "forms/NoteForm/NoteForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeNoteVersion } from "utils/shape/models/noteVersion";
import { NoteCreateProps } from "../types";

export const NoteCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: NoteCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => noteInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<NoteVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<NoteVersion, NoteVersionCreateInput>(noteVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateNote',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    console.log('onsubmit values', values, shapeNoteVersion.create(values))
                    mutationWrapper<NoteVersion, NoteVersionCreateInput>({
                        mutation,
                        input: shapeNoteVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateNoteValues(values, true)}
            >
                {(formik) => <NoteForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
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