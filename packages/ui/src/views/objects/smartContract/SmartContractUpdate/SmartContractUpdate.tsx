import { Grid } from "@mui/material";
import { FindByIdInput, ResourceList, SmartContractVersion, SmartContractVersionUpdateInput } from "@shared/consts";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { smartContractVersionTranslationValidation, smartContractVersionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { smartContractVersionFindOne } from "api/generated/endpoints/smartContractVersion_findOne";
import { smartContractVersionUpdate } from "api/generated/endpoints/smartContractVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeSmartContractVersion } from "utils/shape/models/smartContractVersion";
import { TagShape } from "utils/shape/models/tag";
import { SmartContractUpdateProps } from "../types";

export const SmartContractUpdate = ({
    display = 'page',
    zIndex = 200,
}: SmartContractUpdateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onUpdated } = useUpdateActions<SmartContractVersion>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: smartContractVersion, loading }] = useCustomLazyQuery<SmartContractVersion, FindByIdInput>(smartContractVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useCustomMutation<SmartContractVersion, SmartContractVersionUpdateInput>(smartContractVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: smartContractVersion?.id ?? uuid(),
            translationsUpdate: smartContractVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: '',
                name: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: smartContractVersionValidation.update({}),
        onSubmit: (values) => {
            if (!existing) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                return;
            }
            mutationWrapper<SmartContractVersion, SmartContractVersionUpdateInput>({
                mutation,
                input: shapeSmartContractVersion.update(existing, values),
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
        fields: ['description', 'jsonVariable', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: smartContractVersionTranslationValidation.update({}),
    });

    useEffect(() => {
        setRelationships({
            isComplete: false,
            isPrivate: smartContractVersion?.isPrivate ?? false,
            owner: null,
            parent: null,
            project: null,
        });
        setResourceList(smartContractVersion?.resourceList ?? { id: uuid() } as any);
        setTags(smartContractVersion?.root?.tags ?? []);
        if (smartContractVersion?.translations?.length) {
            setLanguage(getPreferredLanguage(smartContractVersion.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [smartContractVersion, session, setLanguage]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateSmartContract',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipList
                            isEditing={true}
                            objectType={'SmartContract'}
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