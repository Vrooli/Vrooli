import { CommonKey, DUMMY_ID, endpointGetResource, endpointPostResource, endpointPutResource, orDefault, Resource, ResourceCreateInput, ResourceListFor, ResourceUpdateInput, ResourceUsedFor, resourceValidation, Session, userTranslationValidation } from "@local/shared";
import { Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "components/inputs/LinkInput/LinkInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ResourceFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ResourceShape, shapeResource } from "utils/shape/models/resource";
import { ResourceUpsertProps } from "../types";

/** New resources must include a list ID and an index */
export type NewResourceShape = Partial<Omit<Resource, "list">> & {
    index: number,
    list: Partial<Resource["list"]> & ({ id: string } | { listFor: ResourceListFor | `${ResourceListFor}`, listForId: string })
};

export const resourceInitialValues = (
    session: Session | undefined,
    existing: NewResourceShape,
): ResourceShape => ({
    __typename: "Resource" as const,
    id: DUMMY_ID,
    link: "",
    usedFor: ResourceUsedFor.Context,
    ...existing,
    list: {
        __typename: "ResourceList" as const,
        ...existing.list,
        id: existing.list?.id ?? DUMMY_ID,
    },
    translations: orDefault(existing.translations, [{
        __typename: "ResourceTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
    }]),
});

export const transformResourceValues = (values: ResourceShape, existing: ResourceShape, isCreate: boolean) =>
    isCreate ? shapeResource.create(values) : shapeResource.update(existing, values);

export const validateResourceValues = async (values: ResourceShape, existing: ResourceShape, isCreate: boolean) => {
    const transformedValues = transformResourceValues(values, existing, isCreate);
    const validationSchema = resourceValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ResourceForm = forwardRef<BaseFormRef | undefined, ResourceFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    console.log("resource form", isCreate, values);

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
        fields: ["bio"],
        validationSchema: userTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    const [field, , helpers] = useField("translations");

    const foundLinkData = useCallback(({ title, subtitle }: { title: string, subtitle: string }) => {
        // If the user has not entered a name or description, set them to the found values
        if (title.length === 0 || subtitle.length === 0) return;
        const currName = field.value.find((t) => t.language === language)?.name;
        const currDescription = field.value.find((t) => t.language === language)?.description;
        if (currName.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, name: title } : t));
        if (currDescription.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, description: subtitle } : t));
    }, [field, helpers, language]);

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={500}
                ref={ref}
            >
                <Stack direction="column" spacing={2} padding={2}>
                    {/* Language select */}
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />
                    {/* Enter link or search for object */}
                    <LinkInput onObjectData={foundLinkData} />
                    <Selector
                        name="usedFor"
                        options={Object.keys(ResourceUsedFor)}
                        getOptionIcon={(i) => getResourceIcon(i as ResourceUsedFor)}
                        getOptionLabel={(l) => t(l as CommonKey, { count: 2 })}
                        fullWidth
                        label={t("Type")}
                    />
                    <TranslatedTextField
                        fullWidth
                        label={t("NameOptional")}
                        language={language}
                        name="name"
                    />
                    <TranslatedTextField
                        fullWidth
                        label={t("DescriptionOptional")}
                        language={language}
                        multiline
                        maxRows={8}
                        name="description"
                    />
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

export const ResourceUpsert = ({
    isCreate,
    isMutate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ResourceUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Resource, ResourceShape>({
        ...endpointGetResource,
        objectType: "Resource",
        overrideObject: overrideObject as Resource,
        transform: (existing) => resourceInitialValues(session, existing as NewResourceShape),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Resource, ResourceCreateInput, ResourceUpdateInput>({
        display,
        endpointCreate: endpointPostResource,
        endpointUpdate: endpointPutResource,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="resource-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={isCreate ? t("CreateResource") : t("UpdateResource")}
                help={t("ResourceHelp")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    console.log("creating resource", values, transformResourceValues(values, existing, isCreate));
                    if (isMutate) {
                        fetchLazyWrapper<ResourceCreateInput | ResourceUpdateInput, Resource>({
                            fetch,
                            inputs: transformResourceValues(values, existing, isCreate),
                            onSuccess: (data) => { handleCompleted(data); },
                            onCompleted: () => { helpers.setSubmitting(false); },
                        });
                    } else {
                        handleCompleted({
                            ...values,
                            created_at: (existing as Resource)?.created_at ?? new Date().toISOString(),
                            updated_at: (existing as Resource)?.updated_at ?? new Date().toISOString(),
                        } as Resource);
                    }
                }}
                validate={async (values) => await validateResourceValues(values, existing, isCreate)}
            >
                {(formik) => <ResourceForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
