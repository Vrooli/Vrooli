import { DUMMY_ID, Resource, ResourceCreateInput, ResourceShape, ResourceUpdateInput, ResourceUsedFor, Session, TranslationKeyCommon, endpointsResource, noopSubmit, orDefault, resourceValidation, shapeResource, userTranslationValidation } from "@local/shared";
import { Button, Divider, Stack } from "@mui/material";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { LinkInput } from "../../../components/inputs/LinkInput/LinkInput.js";
import { Selector, SelectorBase } from "../../../components/inputs/Selector/Selector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { combineErrorsWithTranslations, getUserLanguages, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { shortcuts } from "../../../utils/navigation/quickActions.js";
import { PubSub } from "../../../utils/pubsub.js";
import { PreSearchItem } from "../../../utils/search/siteToSearch.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ResourceFormProps, ResourceUpsertProps } from "./types.js";

export function resourceInitialValues(
    session: Session | undefined,
    existing: Partial<ResourceShape>,
): ResourceShape {
    let listFor: ResourceShape["list"]["listFor"];
    if (existing.list && "listFor" in existing.list) {
        listFor = existing.list.listFor as ResourceShape["list"]["listFor"];
    } else {
        listFor = { __typename: "RoutineVersion", id: DUMMY_ID };
    }
    return {
        __typename: "Resource" as const,
        id: DUMMY_ID,
        index: 0,
        link: "",
        usedFor: ResourceUsedFor.Context,
        ...existing,
        list: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            ...existing.list,
            listFor,
        },
        translations: orDefault(existing.translations, [{
            __typename: "ResourceTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "",
        }]),
    };
}

export function transformResourceValues(values: ResourceShape, existing: ResourceShape, isCreate: boolean) {
    return isCreate ? shapeResource.create(values) : shapeResource.update(existing, values);
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
        validationSchema: userTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const [field, meta, helpers] = useField("translations");

    const foundLinkData = useCallback(({ title, subtitle }: { title: string, subtitle: string }) => {
        // If the user has not entered a name or description, set them to the found values
        if (title.length === 0 || subtitle.length === 0) return;
        const currName = field.value.find((t) => t.language === language)?.name;
        const currDescription = field.value.find((t) => t.language === language)?.description;
        if (currName.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, name: title } : t));
        if (currDescription.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, description: subtitle } : t));
    }, [field, helpers, language]);

    const { handleCancel, handleCompleted } = useUpsertActions<Resource>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Resource",
        onCancel,
        onCompleted,
        onDeleted,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Resource, ResourceCreateInput, ResourceUpdateInput>({
        isCreate,
        isMutate,
        endpointCreate: endpointsResource.createOne,
        endpointUpdate: endpointsResource.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Resource" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (isMutate) {
            fetchLazyWrapper<ResourceCreateInput | ResourceUpdateInput, Resource>({
                fetch,
                inputs: transformResourceValues(values, existing, isCreate),
                onSuccess: (data) => { handleCompleted(data); },
                onCompleted: () => { props.setSubmitting(false); },
            });
        } else {
            handleCompleted({
                ...values,
                createdAt: (existing as Resource)?.createdAt ?? new Date().toISOString(),
                updatedAt: (existing as Resource)?.updatedAt ?? new Date().toISOString(),
            } as Resource);
        }
    }, [disabled, existing, fetch, handleCompleted, isCreate, isMutate, props, values]);

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

    return (
        <MaybeLargeDialog
            display={display}
            id="resource-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
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
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog >
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
        transform: (existing) => resourceInitialValues(session, existing as ResourceShape),
    });

    async function validateValues(values: ResourceShape) {
        if (!existing) return;
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
