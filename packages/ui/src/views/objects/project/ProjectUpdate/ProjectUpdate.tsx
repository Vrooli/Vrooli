import { Grid } from "@mui/material";
import { FindVersionInput, ProjectVersion, ProjectVersionUpdateInput } from "@shared/consts";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { projectValidation, projectVersionTranslationValidation } from '@shared/validation';
import { projectVersionFindOne } from "api/generated/endpoints/projectVersion_findOne";
import { projectVersionUpdate } from "api/generated/endpoints/projectVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useEffect, useMemo, useState } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeProjectVersion } from "utils/shape/models/projectVersion";
import { TagShape } from "utils/shape/models/tag";
import { ProjectUpdateProps } from "../types";

export const ProjectUpdate = ({
    display = 'page',
    zIndex = 200,
}: ProjectUpdateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onUpdated } = useUpdateActions<ProjectVersion>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: projectVersion, loading }] = useCustomLazyQuery<ProjectVersion, FindVersionInput>(projectVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: projectVersion?.isComplete ?? false,
            isPrivate: projectVersion?.isPrivate ?? false,
            owner: projectVersion?.root?.owner ?? null,
            parent: projectVersion?.root?.parent ?? null,
            project: null,
        });
        setTags(projectVersion?.root?.tags ?? []);
    }, [projectVersion]);

    // Handle update
    const [mutation] = useCustomMutation<ProjectVersion, ProjectVersionUpdateInput>(projectVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: projectVersion?.id ?? uuid(),
            translationsUpdate: projectVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                name: '',
                description: '',
            }],
            versionInfo: {
                versionLabel: projectVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: projectValidation.update({}),
        onSubmit: (values) => {
            if (!existing) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                return;
            }
            mutationWrapper<ProjectVersion, ProjectVersionUpdateInput>({
                mutation,
                input: shapeProjectVersion.update(existing, values),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { helpers.setSubmitting(false) },
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
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: projectVersionTranslationValidation.update({}),
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateProject',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipList
                            isEditing={true}
                            objectType={'Project'}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TranslatedTextField
                            fullWidth
                            label={t('Name')}
                            language={language}
                            name="name"
                        />
                    </Grid>
                    <Grid item xs={12} mb={4}>
                        <TranslatedTextField
                            fullWidth
                            label={t('Description')}
                            language={language}
                            multiline
                            minRows={4}
                            name="name"
                        />
                    </Grid>
                    <Grid item xs={12} mb={4}>
                        <TagSelector
                            handleTagsUpdate={handleTagsUpdate}
                            tags={tags}
                        />
                    </Grid>
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