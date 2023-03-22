import { FindByIdInput, NoteVersion, NoteVersionUpdateInput } from "@shared/consts";
import { DUMMY_ID } from '@shared/uuid';
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
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeNoteVersion } from "utils/shape/models/noteVersion";
import { NoteUpdateProps } from "../types";

export const NoteUpdate = ({
    display = 'page',
    zIndex = 200,
}: NoteUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: noteVersion, loading: isReadLoading }] = useCustomLazyQuery<NoteVersion, FindByIdInput>(noteVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onUpdated } = useUpdateActions<NoteVersion>();
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<NoteVersion, NoteVersionUpdateInput>(noteVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateNote',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'NoteVersion' as const,
                    id: noteVersion?.id ?? DUMMY_ID,
                    isPrivate: noteVersion?.isPrivate ?? false,
                    root: noteVersion?.root ?? {
                        id: DUMMY_ID,
                        isPrivate: false,
                        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
                    },
                    translations: noteVersion?.translations ?? [{
                        id: DUMMY_ID,
                        language: getUserLanguages(session)[0],
                        description: '',
                        name: '',
                        text: '',
                    }],
                    versionLabel: noteVersion?.versionLabel ?? '1.0.0',
                }}
                onSubmit={(values, helpers) => {
                    if (!noteVersion) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadNote', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<NoteVersion, NoteVersionUpdateInput>({
                        mutation,
                        input: shapeNoteVersion.update(noteVersion, values),
                        onSuccess: (data) => { onUpdated(data) },
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
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}