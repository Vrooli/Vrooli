import { Grid, useTheme } from "@mui/material";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { noteVersionValidation, noteVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { defaultRelationships, getUserLanguages, shapeNoteVersion, TagShape, useCreateActions, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { NoteCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, MarkdownInput, RelationshipButtons, TagSelector, TopBar } from "components";
import { uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication";
import { NoteVersion, NoteVersionCreateInput } from "@shared/consts";
import { noteVersionCreate } from "api/generated/endpoints/noteVersion_create";
import { parseSearchParams } from "@shared/route";
import { BaseForm } from "forms";
import { useTranslation } from "react-i18next";

export const NoteCreate = ({
    display = 'page',
    session,
    zIndex = 200,
}: NoteCreateProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { onCancel, onCreated } = useCreateActions<NoteVersion>();

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<NoteVersion, NoteVersionCreateInput>(noteVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            root: {
                id: uuid(),
                isPrivate: false,
                owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
            },
            translations: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
                text: '',
            }],
            versionLabel: '1.0.0',
        },
        validationSchema: noteVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<NoteVersion, NoteVersionCreateInput>({
                mutation,
                input: shapeNoteVersion.create({
                    ...values,
                    root: {
                        ...values.root,
                        isPrivate: relationships.isPrivate,
                    },
                    isPrivate: relationships.isPrivate,
                }),
                onSuccess: (data) => { onCreated(data) },
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
        validationSchema: noteVersionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'CreateNote',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
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
                disabledSubmit={!isLoggedIn}
                display={display}
                errors={translations.errorsWithTranslations}
                isCreate={true}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </>
    )
}