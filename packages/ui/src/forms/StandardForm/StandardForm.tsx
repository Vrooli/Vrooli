import { Box, Stack } from "@mui/material";
import { standardVersionTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { GeneratedInputComponent } from "components/inputs/generated";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { PreviewSwitch } from "components/inputs/PreviewSwitch/PreviewSwitch";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { Selector } from "components/inputs/Selector/Selector";
import { BaseStandardInput } from "components/inputs/standards";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { useFormik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { generateYupSchema } from "forms/generators";
import { FieldData, StandardFormProps } from "forms/types";
import { forwardRef, useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { InputTypeOption, InputTypeOptions } from "utils/consts";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";

export const StandardForm = forwardRef<any, StandardFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

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
    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['description'],
        validationSchema: standardVersionTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    maxWidth: '700px',
                    margin: 'auto',
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={'Standard'}
                        zIndex={zIndex}
                    />
                    <Stack direction="column" spacing={2}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t('Name')}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t('Description')}
                            language={language}
                            multiline
                            minRows={4}
                            maxRows={8}
                            name="description"
                        />
                    </Stack>
                    {/* Standard build/preview */}
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
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <TagSelector />
                    <VersionInput
                        fullWidth
                        versions={routineVersion?.root?.versions ?? []}
                    />
                </Stack>
                <GridSubmitButtons
                    display={display}
                    errors={{
                        ...props.errors,
                        ...translationErrors,
                    }}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    )
})