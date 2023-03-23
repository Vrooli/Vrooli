import { Grid } from "@mui/material";
import { ResourceList, SmartContractVersion, SmartContractVersionCreateInput } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { smartContractVersionTranslationValidation, smartContractVersionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { smartContractVersionCreate } from "api/generated/endpoints/smartContractVersion_create";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { shapeSmartContractVersion } from "utils/shape/models/smartContractVersion";
import { TagShape } from "utils/shape/models/tag";
import { SmartContractCreateProps } from "../types";

export const SmartContractCreate = ({
    display = 'page',
    zIndex = 200,
}: SmartContractCreateProps) => {
    const session = useContext(SessionContext);

    // const formRef = useRef<BaseFormRef>();
    // const { onCancel, onCreated } = useCreateActions<SmartContractVersion>();
    // const [mutation, { loading: isLoading }] = useCustomMutation<SmartContractVersion, SmartContractVersionCreateInput>(smartContractVersionCreate);



    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<SmartContractVersion, SmartContractVersionCreateInput>(smartContractVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: '',
                name: '',
            }]
        },
        validationSchema: smartContractVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<SmartContractVersion, SmartContractVersionCreateInput>({
                mutation,
                input: shapeSmartContractVersion.create(values),
                onSuccess: (data) => { onCreated(data) },
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
        formikField: 'translationsCreate',
        validationSchema: smartContractVersionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateSmartContract',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
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
                            translations={formik.values.translationsCreate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* TODO */}
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
                </Grid>
            </BaseForm>
        </>
    )
}