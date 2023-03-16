import { Grid, useTheme } from "@mui/material";
import { FindVersionInput, NoteVersion, NoteVersionUpdateInput } from "@shared/consts";
import { DUMMY_ID } from '@shared/uuid';
import { noteVersionTranslationValidation, noteVersionValidation } from '@shared/validation';
import { noteVersionFindOne } from "api/generated/endpoints/noteVersion_findOne";
import { noteVersionUpdate } from "api/generated/endpoints/noteVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { RelationshipsObject } from "components/inputs/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { defaultRelationships } from "utils/defaults/relationships";
import { getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { shapeNoteVersion } from "utils/shape/models/noteVersion";
import { TagShape } from "utils/shape/models/tag";
import { NoteUpdateProps } from "../types";

export const NoteUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: NoteUpdateProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    
    const { onCancel, onUpdated } = useUpdateActions<NoteVersion>();

    // Fetch existing data
    const fetchParams = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: noteVersion, loading }] = useCustomLazyQuery<NoteVersion, FindVersionInput>(noteVersionFindOne);
    useEffect(() => { Object.keys(fetchParams).length && getData({ variables: fetchParams }) }, [fetchParams, getData]);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    console.log('rendering note update', noteVersion)
    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useCustomMutation<NoteVersion, NoteVersionUpdateInput>(noteVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: noteVersion?.id ?? DUMMY_ID,
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
        },
        enableReinitialize: true,
        validationSchema: noteVersionValidation.update({}),
        onSubmit: (values) => {
            if (!noteVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadNote', severity: 'Error' });
                return;
            }
            mutationWrapper<NoteVersion, NoteVersionUpdateInput>({
                mutation,
                input: shapeNoteVersion.update(noteVersion, {
                    ...values,
                    root: {
                        ...values.root,
                        isPrivate: relationships.isPrivate,
                    },
                    isPrivate: relationships.isPrivate,
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
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
        formikField: 'translations',
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
                            translations={formik.values.translations}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <MarkdownInput
                            id="text"
                            placeholder={t(`PleaseBeNice`)}
                            value={translations.text}
                            minRows={3}
                            onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                            error={translations.text.length > 0 && Boolean(translations.errorText)}
                            helperText={translations.text.length > 0 ? translations.errorText : ''}
                            sxs={{
                                bar: {
                                    borderRadius: 0,
                                    background: palette.primary.main,
                                },
                                textArea: {
                                    borderRadius: 0,
                                    resize: 'none',
                                    minHeight: '100vh',
                                    background: palette.background.paper,
                                }
                            }}
                        />
                    </Grid>
                </Grid>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </>
    )
}