import { CodeLanguage, CodeShape, CodeType, CodeVersion, CodeVersionCreateInput, CodeVersionShape, CodeVersionUpdateInput, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, Session, codeVersionTranslationValidation, codeVersionValidation, endpointsCodeVersion, noopSubmit, orDefault, shapeCodeVersion } from "@local/shared";
import { Button, Divider } from "@mui/material";
import { useSubmitHelper } from "api/fetchWrapper.js";
import { AutoFillButton } from "components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton.js";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse.js";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { CodeInput } from "components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput.js";
import { TagSelector } from "components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput.js";
import { VersionInput } from "components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "components/lists/ResourceList/ResourceList.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "hooks/forms.js";
import { UseAutoFillProps, createUpdatedTranslations, getAutoFillTranslationData, useAutoFill } from "hooks/tasks.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useTranslatedFields } from "hooks/useTranslatedFields.js";
import { useUpsertFetch } from "hooks/useUpsertFetch.js";
import { HelpIcon } from "icons/common.js";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools.js";
import { validateFormValues } from "utils/validateFormValues.js";
import { SessionContext } from "../../../contexts.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { DataConverterFormProps, DataConverterUpsertProps } from "./types.js";

export function dataConverterInitialValues(
    session: Session | undefined,
    existing?: Partial<CodeVersion> | undefined,
): CodeVersionShape {
    return {
        __typename: "CodeVersion" as const,
        id: DUMMY_ID,
        calledByRoutineVersionsCount: 0,
        codeLanguage: CodeLanguage.Haskell,
        codeType: CodeType.DataConvert,
        content: "",
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        resourceList: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "CodeVersion" as const,
                id: DUMMY_ID,
            },
        },
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Code" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            permissions: JSON.stringify({}),
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "CodeVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            jsonVariable: "",
            name: "",
        }]),
    };
}

function transformDataConverterVersionValues(values: CodeVersionShape, existing: CodeVersionShape, isCreate: boolean) {
    return isCreate ? shapeCodeVersion.create(values) : shapeCodeVersion.update(existing, values);
}

/** Code to display when an example is requested */
const exampleCode = `
/**
 * Converts a comma-separated string of numbers into an array of integers.
 * 
 * This function attempts to parse each element of the input string, 
 * which should be numbers separated by commas, into integers. If any 
 * part of the string cannot be converted to an integer, an error object 
 * is returned instead.
 *
 * @param {string} input - A string of numbers separated by commas.
 * @returns {number[]|Object} - Returns an array of integers if successful, 
 *                              or an object with an error message if the input is invalid.
 * 
 * @example
 * parseList("1, 2, 3, 4"); // returns [1, 2, 3, 4]
 * parseList("1, two, 3");  // returns { error: "Invalid input format" }
 */
function parseList(input) {
    try {
        // Parse the input string into an array of strings split by commas
        const stringArray = input.split(',');

        // Map each string element to an integer
        const numberArray = stringArray.map(element => parseInt(element.trim(), 10));

        // Return the array of numbers
        return numberArray;
    } catch (error) {
        // Return an error message if the input is not as expected
        return { error: "Invalid input format" };
    }
}
`.trim();

const codeLimitTo = [CodeLanguage.Javascript] as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;
const codeCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;

function DataConverterForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    versions,
    ...props
}: DataConverterFormProps) {
    const session = useContext(SessionContext);
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
        validationSchema: codeVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "CodeVersion", id: values.id } as const;
    }, [values]);

    const { handleCancel, handleCompleted } = useUpsertActions<CodeVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "CodeVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<CodeVersion, CodeVersionCreateInput, CodeVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsCodeVersion.createOne,
        endpointUpdate: endpointsCodeVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "CodeVersion" });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            content: values.content,
            isPrivate: values.isPrivate,
            version: values.versionLabel,
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["name", "description"]);
        delete rest.id;
        const content = typeof rest.content === "string" ? rest.content : values.content;
        const isPrivate = typeof rest.isPrivate === "boolean" ? rest.isPrivate : values.isPrivate;
        const versionLabel = typeof rest.version === "string" ? rest.version : values.versionLabel;
        const updatedValues = {
            ...values,
            content,
            isPrivate,
            translations: updatedTranslations,
            versionLabel,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.DataConverterAdd : LlmTask.DataConverterUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<CodeVersionCreateInput | CodeVersionUpdateInput, CodeVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformDataConverterVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const [, , codeLanguageHelpers] = useField<CodeLanguage>("codeLanguage");
    const [, , contentHelpers] = useField<string>("content");
    function showExample() {
        // We only have an example for JavaScript, so switch to that
        codeLanguageHelpers.setValue(CodeLanguage.Javascript);
        // Set value to hard-coded example
        contentHelpers.setValue(exampleCode);
    }

    return (
        <MaybeLargeDialog
            display={display}
            id="code-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateDataConverter" : "UpdateDataConverter")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.DataConverter}"`}
                text="Search existing codes"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Code"}
                            sx={relationshipListStyle}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={resourceListParent}
                            sxs={resourceListStyle}
                        />
                        <FormSection sx={formSectionStyle}>
                            <TranslatedTextInput
                                fullWidth
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="description"
                                maxChars={2048}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("DescriptionPlaceholder")}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                flexDirection="row-reverse"
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                            />
                        </FormSection>
                        <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={{ marginBottom: 2 }}
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse
                        title="Code"
                        titleVariant="h4"
                        isOpen={true}
                        toTheRight={
                            <Button
                                variant="outlined"
                                onClick={showExample}
                                startIcon={<HelpIcon />}
                                sx={exampleButtonStyle}
                            >
                                Show example
                            </Button>
                        }
                        sxs={codeCollapseStyle}
                    >
                        <CodeInput
                            disabled={false}
                            limitTo={codeLimitTo}
                            name="content"
                        />
                    </ContentCollapse>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </MaybeLargeDialog>
    );
}

export function DataConverterUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: DataConverterUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<CodeVersion, CodeVersionShape>({
        ...endpointsCodeVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "CodeVersion",
        overrideObject,
        transform: (existing) => dataConverterInitialValues(session, existing),
    });

    async function validateValues(values: CodeVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformDataConverterVersionValues, codeVersionValidation);
    }

    const versions = useMemo(function versionsMemo() {
        return (existing?.root as CodeShape)?.versions?.map(v => v.versionLabel) ?? [];
    }, [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <DataConverterForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={versions}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
