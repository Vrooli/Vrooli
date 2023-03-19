import { Stack, TextField, useTheme } from "@mui/material";
import { SearchIcon } from "@shared/icons";
import { userTranslationValidation } from "@shared/validation";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { Field, FormikProps } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { ViewDisplayType } from "views/types";

interface ResourceFormProps extends FormikProps<any> {
    display: ViewDisplayType;
    index: number;
    isLoading: boolean;
    onCancel: () => void;
    ref: React.RefObject<any>;
    zIndex: number;
}

export const ResourceForm = forwardRef<any, ResourceFormProps>(({
    display,
    dirty,
    index,
    isLoading,
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

    return (
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
                        // onClick={openSearch}
                        background={palette.secondary.main}
                        sx={{
                            borderRadius: '0 5px 5px 0',
                            height: '56px',
                        }}>
                        <SearchIcon />
                    </ColorIconButton>
                </Stack>
                {/* Select resource type */}
                {/* <FormControl fullWidth>
                    <InputLabel id="resource-type-label">{t('Type')}</InputLabel>
                    <Select
                        labelId="resource-type-label"
                        id="usedFor"
                        value={formik.values.usedFor}
                        onChange={(e) => formik.setFieldValue('usedFor', e.target.value)}
                        sx={{
                            '& .MuiSelect-select': {
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                textAlign: 'left',
                            },
                        }}
                    >
                        {(Object.keys(ResourceUsedFor) as Array<keyof typeof ResourceUsedFor>).map((usedFor) => {
                            const Icon = getResourceIcon(usedFor as ResourceUsedFor);
                            return (
                                <MenuItem key={usedFor} value={usedFor}>
                                    <ListItemIcon>
                                        <Icon fill={palette.background.textSecondary} />
                                    </ListItemIcon>
                                    <ListItemText>{t(usedFor as CommonKey, { count: 2 })}</ListItemText>
                                </MenuItem>
                            )
                        })}
                    </Select>
                </FormControl> */}
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
                isCreate={index < 0}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    )
})