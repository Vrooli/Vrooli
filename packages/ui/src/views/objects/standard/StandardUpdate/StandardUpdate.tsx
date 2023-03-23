import { Grid, TextField } from "@mui/material";
import { FindVersionInput, ResourceList, StandardVersion, StandardVersionUpdateInput } from "@shared/consts";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { standardVersionTranslationValidation, standardVersionValidation } from "@shared/validation";
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion_findOne";
import { standardVersionUpdate } from "api/generated/endpoints/standardVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeStandardVersion } from "utils/shape/models/standardVersion";
import { TagShape } from "utils/shape/models/tag";
import { StandardUpdateProps } from "../types";

export const StandardUpdate = ({
    display = 'page',
    zIndex = 200,
}: StandardUpdateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onUpdated } = useUpdateActions<StandardVersion>();

    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: standardVersion, loading }] = useCustomLazyQuery<StandardVersion, FindVersionInput>(standardVersionFindOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, urlData])

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standardVersion?.isPrivate ?? false,
            owner: standardVersion?.root?.owner ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standardVersion?.resourceList ?? { id: uuid() } as any);
        setTags(standardVersion?.root?.tags ?? []);
    }, [standardVersion]);

    // Handle update
    const [mutation] = useCustomMutation<StandardVersion, StandardVersionUpdateInput>(standardVersionUpdate);
    const formik = useFormik({
        initialValues: {
            translationsUpdate: standardVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
            versionInfo: {
                versionLabel: standardVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: standardVersionValidation.update({}),
        onSubmit: (values) => {
            if (!existing) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                return;
            }
            mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                mutation,
                input: shapeStandardVersion.update(existing, values),
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
        fields: ['description'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: standardVersionTranslationValidation.update({}),
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateStandard',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipList
                            isEditing={true}
                            objectType={'Standard'}
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
                    <Grid item xs={12} mb={4}>
                        <TextField
                            fullWidth
                            id="description"
                            name="description"
                            label="description"
                            multiline
                            minRows={4}
                            value={translations.description}
                            onBlur={onTranslationBlur}
                            onChange={onTranslationChange}
                            error={translations.touchedDescription && Boolean(translations.errorDescription)}
                            helperText={translations.touchedDescription && translations.errorDescription}
                        />
                    </Grid>
                    {/* TODO versioning */}
                    <Grid item xs={12}>
                        <ResourceListHorizontal
                            title={'Resources'}
                            list={resourceList}
                            canUpdate={true}
                            handleUpdate={handleResourcesUpdate}
                            loading={loading}
                            mutate={false}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12} marginBottom={4}>
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