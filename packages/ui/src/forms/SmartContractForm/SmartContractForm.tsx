import { DUMMY_ID, orDefault, Session, SmartContractVersion, smartContractVersionTranslationValidation, smartContractVersionValidation } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { SmartContractFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeSmartContractVersion, SmartContractVersionShape } from "utils/shape/models/smartContractVersion";

export const smartContractInitialValues = (
    session: Session | undefined,
    existing?: SmartContractVersion | undefined
): SmartContractVersionShape => ({
    __typename: 'SmartContractVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    content: '',
    contractType: '',
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'SmartContract' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    versionLabel: '1.0.0',
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: 'SmartContractVersionTranslation' as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        jsonVariable: '',
        name: '',
    }]),
});

export const transformSmartContractValues = (values: SmartContractVersionShape, existing?: SmartContractVersionShape) => {
    return existing === undefined
        ? shapeSmartContractVersion.create(values)
        : shapeSmartContractVersion.update(existing, values)
}

export const validateSmartContractValues = async (values: SmartContractVersionShape, existing?: SmartContractVersionShape) => {
    const transformedValues = transformSmartContractValues(values, existing);
    const validationSchema = smartContractVersionValidation[existing === undefined ? 'create' : 'update']({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const SmartContractForm = forwardRef<any, SmartContractFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
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
        fields: ['description', 'jsonVariable', 'name'],
        validationSchema: smartContractVersionTranslationValidation[isCreate ? 'create' : 'update']({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    width: 'min(700px, 100vw - 16px)',
                    margin: 'auto',
                    paddingLeft: 'env(safe-area-inset-left)',
                    paddingRight: 'env(safe-area-inset-right)',
                    paddingBottom: 'calc(64px + env(safe-area-inset-bottom))',
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
                </Stack>
                {/* TODO */}
                <VersionInput
                    fullWidth
                    versions={versions}
                />
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
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