import { AutoFillResult, CodeLanguage, CodeShape, CodeType, CodeVersion, CodeVersionCreateInput, CodeVersionShape, CodeVersionUpdateInput, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, Session, codeVersionTranslationValidation, codeVersionValidation, endpointGetCodeVersion, endpointPostCodeVersion, endpointPutCodeVersion, noopSubmit, orDefault, shapeCodeVersion } from "@local/shared";
import { Button, Divider } from "@mui/material";
import { useSubmitHelper } from "api";
import { AutoFillButton } from "components/buttons/AutoFillButton/AutoFillButton";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { getAutoFillTranslationData, useAutoFill } from "hooks/useAutoFill";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { HelpIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateFormValues } from "utils/validateFormValues";
import { CodeFormProps, CodeUpsertProps } from "../types";

export function codeInitialValues(
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

function transformCodeVersionValues(values: CodeVersionShape, existing: CodeVersionShape, isCreate: boolean) {
    return isCreate ? shapeCodeVersion.create(values) : shapeCodeVersion.update(existing, values);
}

/** Code to display when an example is requested */
const exampleCode = `
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

function CodeForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    versions,
    ...props
}: CodeFormProps) {
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
        endpointCreate: endpointPostCodeVersion,
        endpointUpdate: endpointPutCodeVersion,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "CodeVersion" });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            //TODO
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback({ data }: AutoFillResult) {
        const originalValues = { ...values };
        const updatedValues = {} as any; //TODO
        console.log("in shapeAutoFillResult", language, data, originalValues, updatedValues);
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
        inputs: transformCodeVersionValues(values, existing, isCreate),
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
                title={t(isCreate ? "CreateCode" : "UpdateCode")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Code}"`}
                text="Search existing codes"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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
                            parent={{ __typename: "CodeVersion", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
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
                            <>
                                <Button
                                    variant="outlined"
                                    onClick={showExample}
                                    startIcon={<HelpIcon />}
                                    sx={{ marginLeft: 2 }}
                                >
                                    Show example
                                </Button>
                                {/* <IconButton
                                    onClick={toggleFullscreen}
                                    sx={{ marginLeft: 2 }}
                                >
                                    {isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
                                </IconButton> */}
                            </>
                        }
                        sxs={{ titleContainer: { marginBottom: 1 } }}
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

export function CodeUpsert({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: CodeUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<CodeVersion, CodeVersionShape>({
        ...endpointGetCodeVersion,
        isCreate,
        objectType: "CodeVersion",
        overrideObject,
        transform: (existing) => codeInitialValues(session, existing),
    });

    async function validateValues(values: CodeVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformCodeVersionValues, codeVersionValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <CodeForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={(existing?.root as CodeShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
