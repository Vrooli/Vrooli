import { Stack, useTheme } from "@mui/material";
import { Session, SmartContractVersion } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { smartContractVersionTranslationValidation, smartContractVersionValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { SmartContractFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
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
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        jsonVariable: '',
        name: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});

export const transformSmartContractValues = (o: SmartContractVersionShape, u?: SmartContractVersionShape) => {
    return u === undefined
        ? shapeSmartContractVersion.create(o)
        : shapeSmartContractVersion.update(o, u)
}

export const validateSmartContractValues = async (values: SmartContractVersionShape, isCreate: boolean) => {
    const transformedValues = transformSmartContractValues(values);
    const validationSchema = isCreate
        ? smartContractVersionValidation.create({})
        : smartContractVersionValidation.update({});
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
        validationSchema: smartContractVersionTranslationValidation.update({}),
    });

    return (
        <>
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
                </Stack>
                {/* TODO */}
                <VersionInput
                    fullWidth
                    versions={versions}
                />
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