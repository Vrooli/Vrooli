import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import Divider from "@mui/material/Divider";
import Stack from "@mui/material/Stack";
import { DUMMY_ID, endpointsResource, resourceFormConfig, noopSubmit, orDefault, resourceValidation, userTranslationValidation, ResourceUsedFor, type Resource, type ResourceCreateInput, type ResourceShape, type ResourceUpdateInput, type Session, type TranslationKeyCommon } from "@vrooli/shared";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { LinkInput } from "../../../components/inputs/LinkInput/LinkInput.js";
import { Selector, SelectorBase } from "../../../components/inputs/Selector/Selector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { getUserLanguages, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { shortcuts } from "../../../utils/navigation/quickActions.js";
import { type PreSearchItem } from "../../../utils/search/siteToSearch.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { type ResourceFormProps, type ResourceUpsertProps } from "./types.js";

export function transformResourceValues(values: ResourceShape, existing: ResourceShape, isCreate: boolean) {
    return isCreate ? resourceFormConfig.transformations.shapeToInput.create?.(values) : resourceFormConfig.transformations.shapeToInput.update?.(existing, values);
}

function ResourceForm({
    disabled,
    display,
    existing,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: ResourceFormProps) {
    const { t } = useTranslation();
    const isMobile = useIsMobile();

    // Use the standardized form hook
    const {
        session,
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
        language,
        languages,
        handleAddLanguage,
        handleDeleteLanguage,
        setLanguage,
        translationErrors,
    } = useStandardUpsertForm({
        objectType: "Resource",
        validation: resourceValidation,
        translationValidation: userTranslationValidation,
        transformFunction: transformResourceValues,
        endpoints: {
            create: resourceFormConfig.endpoints.createOne,
            update: resourceFormConfig.endpoints.updateOne,
        },
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isMutate,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate: props.handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const [field, meta, helpers] = useField("translations");

    const foundLinkData = useCallback(({ title, subtitle }: { title: string, subtitle: string }) => {
        // If the user has not entered a name or description, set them to the found values
        if (title.length === 0 || subtitle.length === 0) return;
        
        // Ensure field.value is an array before calling find
        if (!Array.isArray(field.value)) return;
        
        const currName = field.value.find((t) => t.language === language)?.name;
        const currDescription = field.value.find((t) => t.language === language)?.description;
        
        // Ensure currName and currDescription are strings before checking length
        if (currName != null && currName.length === 0) {
            helpers.setValue(field.value.map((t) => t.language === language ? { ...t, name: title } : t));
        }
        if (currDescription != null && currDescription.length === 0) {
            helpers.setValue(field.value.map((t) => t.language === language ? { ...t, description: subtitle } : t));
        }
    }, [field, helpers, language]);


    type ResourceType = "shortcut" | "link";
    const [resourceType, setResourceType] = useState<ResourceType>("link");
    const [selectedShortcut, setSelectedShortCut] = useState<PreSearchItem | null>(null);
    const handleResourceTypeChange = useCallback((updatedResourceType: ResourceType) => {
        setSelectedShortCut(null);
        setResourceType(updatedResourceType);
    }, []);
    const handleShortcutChange = useCallback((shortcut: PreSearchItem | null) => {
        console.log("handleshortcutchange", shortcut);
        setSelectedShortCut(shortcut);
        if (shortcut) {
            // props.setFieldValue("link", shortcut.value);
            props.handleChange({ target: { name: "link", value: `${window.location.origin}${shortcut.value}` } });
            // Set name
            handleTranslationChange(field, meta, helpers, { target: { name: "name", value: t(shortcut.label as TranslationKeyCommon, { count: 1 }) } }, language);
        }
    }, [field, meta, helpers, props, t, language]);

    const dialogContent = (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={isCreate ? t("AddResource") : t("UpdateResource")}
                help={t("ResourceHelp")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer>
                    <FormSection variant="transparent">
                        <Stack direction="row" spacing={2}>
                            <Button
                                disabled={disabled || resourceType === "link"}
                                fullWidth
                                onClick={() => { handleResourceTypeChange("link"); }}
                                variant={"outlined"}
                                startIcon={resourceType === "link" ? <IconCommon name="Complete" /> : undefined}
                            >
                                {t("Link", { count: 1 })}
                            </Button>
                            <Button
                                disabled={disabled || resourceType === "shortcut"}
                                fullWidth
                                onClick={() => { handleResourceTypeChange("shortcut"); }}
                                variant={"outlined"}
                                startIcon={resourceType === "shortcut" ? <IconCommon name="Complete" /> : undefined}
                            >
                                {t("Shortcut", { count: 1 })}
                            </Button>
                        </Stack>
                        {resourceType === "shortcut" && <>
                            <SelectorBase
                                name="shortcut"
                                value={selectedShortcut}
                                onChange={(newValue) => { handleShortcutChange(newValue); }}
                                options={shortcuts}
                                getOptionLabel={(shortcut) => t(shortcut.label as TranslationKeyCommon, { count: 1 })}
                                fullWidth
                                label={t("SelectShortcut")}
                            />
                        </>}
                        {resourceType === "link" && <>
                            {/* Enter link or search for object */}
                            <LinkInput
                                autoFocus
                                name="link"
                                onObjectData={foundLinkData}
                                tabIndex={0}
                            />
                            <Selector
                                name="usedFor"
                                options={Object.keys(ResourceUsedFor)}
                                getOptionIcon={(i) => getResourceIcon({ usedFor: i as ResourceUsedFor })}
                                getOptionLabel={(l) => t(l as TranslationKeyCommon, { count: 2 })}
                                fullWidth
                                label={t("Type")}
                            />
                        </>}
                    </FormSection>
                    <Divider />
                    <FormSection variant="transparent">
                        <TranslatedTextInput
                            fullWidth
                            isRequired={false}
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextInput
                            fullWidth
                            isRequired={false}
                            label={t("Description")}
                            language={language}
                            multiline
                            minRows={2}
                            maxRows={8}
                            name="description"
                        />
                        {/* Language select */}
                        {/*<LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />*/}
                    </FormSection>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={translationErrors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </>
    );

    return isMobile ? (
        <Dialog
            id="resource-upsert-dialog"
            open={isOpen}
            onClose={onClose}
            fullScreen
        >
            {dialogContent}
        </Dialog>
    ) : (
        <Dialog
            id="resource-upsert-dialog"
            open={isOpen}
            onClose={onClose}
            maxWidth="sm"
            fullWidth
        >
            {dialogContent}
        </Dialog>
    );
}

export function ResourceUpsert({
    display,
    isCreate,
    isMutate,
    isOpen,
    overrideObject,
    ...props
}: ResourceUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Resource, ResourceShape>({
        ...endpointsResource.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Resource",
        overrideObject: overrideObject as Resource,
        transform: (existing) => resourceFormConfig.transformations.getInitialValues(session, existing as ResourceShape),
    });

    // Validation for the wrapper Formik
    async function validateValues(values: ResourceShape) {
        return await validateFormValues(values, existing, isCreate, transformResourceValues, resourceValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ResourceForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isMutate={isMutate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
