import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { NoteUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { noteVersionValidation, noteVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, parseSingleItemUrl, PubSub, shapeNoteVersion, TagShape, usePromptBeforeUnload, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, RelationshipButtons, TagSelector, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, NoteVersion, NoteVersionUpdateInput, ResourceList } from "@shared/consts";
import { noteVersionFindOne } from "api/generated/endpoints/noteVersion_findOne";
import { noteVersionUpdate } from "api/generated/endpoints/noteVersion_update";
import { BaseForm } from "forms";

export const NoteUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: NoteUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<NoteVersion>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: noteVersion, loading }] = useCustomLazyQuery<NoteVersion, FindByIdInput>(noteVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useCustomMutation<NoteVersion, NoteVersionUpdateInput>(noteVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: noteVersion?.id ?? uuid(),
            translationsUpdate: noteVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
                text: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: noteVersionValidation.update({}),
        onSubmit: (values) => {
            if (!noteVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadNote', severity: 'Error' });
                return;
            }
            // mutationWrapper<NoteVersion, NoteVersionUpdateInput>({
            //     mutation,
            //     input: shapeNoteVersion.update(noteVersion, {
            //         id: noteVersion.id,
            //         isOpenToNewMembers: values.isOpenToNewMembers,
            //         isPrivate: relationships.isPrivate,
            //         tags: tags,
            //         translations: values.translationsUpdate,
            //     }),
            //     onSuccess: (data) => { onUpdated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['description', 'name', 'text'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: noteVersionTranslationValidation.update({}),
    });

    useEffect(() => {
        setRelationships({
            isComplete: false,
            isPrivate: noteVersion?.isPrivate ?? false,
            owner: null,
            parent: null,
            project: null,
        });
        setTags(noteVersion?.root?.tags ?? []);
        if (noteVersion?.translations?.length) {
            setLanguage(getPreferredLanguage(noteVersion.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [noteVersion, session, setLanguage]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateNote',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
                            isEditing={true}
                            objectType={'Note'}
                            onRelationshipsChange={onRelationshipsChange}
                            relationships={relationships}
                            session={session}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            session={session}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* TODO */}
                    <GridSubmitButtons
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}