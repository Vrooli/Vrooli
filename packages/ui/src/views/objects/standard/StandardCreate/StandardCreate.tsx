import { Box, Grid, TextField } from "@mui/material";
import { ResourceList, StandardVersion, StandardVersionCreateInput } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { standardVersionTranslationValidation, standardVersionValidation } from "@shared/validation";
import { standardVersionCreate } from "api/generated/endpoints/standardVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { GeneratedInputComponent } from "components/inputs/generated";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { PreviewSwitch } from "components/inputs/PreviewSwitch/PreviewSwitch";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { Selector } from "components/inputs/Selector/Selector";
import { BaseStandardInput } from "components/inputs/standards";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { generateYupSchema } from "forms/generators";
import { FieldData } from "forms/types";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { InputTypeOption, InputTypeOptions } from "utils/consts";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { shapeStandardVersion } from "utils/shape/models/standardVersion";
import { TagShape } from "utils/shape/models/tag";
import { StandardCreateProps } from "../types";

export const StandardCreate = ({
    display = 'page',
    zIndex = 200,
}: StandardCreateProps) => {
    const session = useContext(SessionContext);

    // const formRef = useRef<BaseFormRef>();
    // const { onCancel, onCreated } = useCreateActions<StandardVersion>();
    // const [mutation, { loading: isLoading }] = useCustomMutation<StandardVersion, StandardVersionCreateInput>(standardVersionCreate);




    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        setInputType(selected);
    }, []);

    // Handle standard schema
    const [schema, setSchema] = useState<FieldData | null>(null);
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        setSchema(schema);
    }, []);
    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);

    const previewFormik = useFormik({
        initialValues: {
            preview: schema?.props?.defaultValue,
        },
        enableReinitialize: true,
        validationSchema: schema ? generateYupSchema({
            fields: [schema],
        }) : undefined,
        onSubmit: () => { },
    });

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
    const [mutation] = useCustomMutation<StandardVersion, StandardVersionCreateInput>(standardVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            default: '',
            name: '',
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
            versionInfo: {
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: standardVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<StandardVersion, StandardVersionCreateInput>({
                mutation,
                input: shapeStandardVersion.create(values),
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
        fields: ['description'],
        formik,
        formikField: 'translationsCreate',
        validationSchema: standardVersionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateStandard',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
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
                            translations={formik.values.translationsCreate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="name"
                            name="name"
                            label="Name"
                            value={formik.values.name}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.name && Boolean(formik.errors.name)}
                            helperText={formik.touched.name && formik.errors.name}
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
                    {/* <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="version"
                        name="version"
                        label="version"
                        value={formik.values.version}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.version && Boolean(formik.errors.version)}
                        helperText={formik.touched.version && formik.errors.version}
                    />
                </Grid> */}
                    {/* Standard build/preview */}
                    <Grid item xs={12}>
                        <PreviewSwitch
                            isPreviewOn={isPreviewOn}
                            onChange={onPreviewChange}
                            sx={{
                                marginBottom: 2
                            }}
                        />
                        {
                            isPreviewOn ?
                                (schema && <GeneratedInputComponent
                                    disabled={true}
                                    fieldData={schema}
                                    onUpload={() => { }}
                                    zIndex={zIndex}
                                />) :
                                <Box>
                                    <Selector
                                        fullWidth
                                        options={InputTypeOptions}
                                        selected={inputType}
                                        handleChange={handleInputTypeSelect}
                                        getOptionLabel={(option: InputTypeOption) => option.label}
                                        inputAriaLabel='input-type-selector'
                                        label="Type"
                                        sx={{ marginBottom: 2 }}
                                    />
                                    <BaseStandardInput
                                        fieldName="preview"
                                        inputType={inputType.value}
                                        isEditing={true}
                                        schema={schema}
                                        onChange={handleSchemaUpdate}
                                        storageKey={schemaKey}
                                    />
                                </Box>
                        }
                    </Grid>
                    <Grid item xs={12}>
                        <ResourceListHorizontal
                            title={'Resources'}
                            list={resourceList}
                            canUpdate={true}
                            handleUpdate={handleResourcesUpdate}
                            loading={false}
                            mutate={false}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12} mb={4}>
                        <TagSelector
                            handleTagsUpdate={handleTagsUpdate}
                            tags={tags}
                        />
                    </Grid>
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