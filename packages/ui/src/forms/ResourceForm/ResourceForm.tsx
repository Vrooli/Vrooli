import { Stack, TextField, useTheme } from "@mui/material";
import { ResourceUsedFor } from "@shared/consts";
import { SearchIcon } from "@shared/icons";
import { CommonKey } from "@shared/translations";
import { userTranslationValidation } from "@shared/validation";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { Selector } from "components/inputs/Selector/Selector";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ResourceFormProps } from "forms/types";
import { forwardRef, useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";

export const ResourceForm = forwardRef<any, ResourceFormProps>(({
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
    const { palette } = useTheme();
    const { t } = useTranslation();

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
        fields: ['bio'],
        validationSchema: userTranslationValidation.update({}),
    });

    // Search dialog to find routines, organizations, etc. to link to
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true) }, []);
    const closeSearch = useCallback((selectedUrl?: string) => {
        setSearchOpen(false);
        if (selectedUrl) {
            props.setFieldValue('link', selectedUrl);
        }
    }, [props]);

    return (
        <>
            {/* Search dialog */}
            <FindObjectDialog
                isOpen={searchOpen}
                handleClose={closeSearch}
                zIndex={zIndex + 1}
            />
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    paddingBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={2} paddingTop={2}>
                    {/* Language select */}
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                        zIndex={zIndex + 1}
                    />
                    {/* Enter link or search for object */}
                    <Stack direction="row" spacing={0}>
                        <Field
                            fullWidth
                            name="link"
                            label={t('Link')}
                            as={TextField}
                            sx={{
                                '& .MuiInputBase-root': {
                                    borderRadius: '5px 0 0 5px',
                                }
                            }}
                        />
                        <ColorIconButton
                            aria-label='find URL'
                            onClick={openSearch}
                            background={palette.secondary.main}
                            sx={{
                                borderRadius: '0 5px 5px 0',
                                height: '56px',
                            }}>
                            <SearchIcon />
                        </ColorIconButton>
                    </Stack>
                    {/* Select resource type */}
                    <Selector
                        name="usedFor"
                        options={Object.keys(ResourceUsedFor)}
                        getOptionIcon={(i) => getResourceIcon(i as ResourceUsedFor)}
                        getOptionLabel={(l) => t(l as CommonKey, { count: 2 })}
                        fullWidth
                        label={t('Type')}
                    />
                    {/* Enter name */}
                    <Field
                        fullWidth
                        name="name"
                        label={t('Name')}
                        helperText={t('NameOptional')}
                        as={TextField}
                    />
                    {/* Enter description */}
                    <Field
                        fullWidth
                        name="description"
                        label={t('Description')}
                        helperText={t('DescriptionOptional')}
                        multiline
                        maxRows={8}
                        as={TextField}
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