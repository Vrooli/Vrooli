import { CommonKey, DUMMY_ID, endpointGetResource, endpointPostResource, endpointPutResource, noopSubmit, orDefault, Resource, ResourceCreateInput, ResourceListFor, ResourceUpdateInput, ResourceUsedFor, resourceValidation, Session, userTranslationValidation } from "@local/shared";
import { Divider, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "components/inputs/LinkInput/LinkInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { ResourceShape, shapeResource } from "utils/shape/models/resource";
import { validateFormValues } from "utils/validateFormValues";
import { ResourceFormProps, ResourceUpsertProps } from "../types";

/** New resources must include a list ID and an index */
export type NewResourceShape = Partial<Omit<Resource, "list">> & {
    index: number,
    list: Partial<Resource["list"]> & ({ id: string } | { listForType: ResourceListFor | `${ResourceListFor}`, listForId: string })
};

export const resourceInitialValues = (
    session: Session | undefined,
    existing: NewResourceShape,
): ResourceShape => {
    let listFor: { __typename: ResourceListFor, id: string };
    if ("listForId" in existing.list && "listForType" in existing.list) {
        listFor = {
            __typename: existing.list.listForType as ResourceListFor,
            id: existing.list.listForId,
        };
    } else if ("listFor" in existing.list) {
        listFor = existing.list.listFor as { __typename: ResourceListFor, id: string };
    } else {
        listFor = {
            __typename: "RoutineVersion" as ResourceListFor,
            id: DUMMY_ID,
        };
    }
    return {
        __typename: "Resource" as const,
        id: DUMMY_ID,
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
};

export const transformResourceValues = (values: ResourceShape, existing: ResourceShape, isCreate: boolean) =>
    isCreate ? shapeResource.create(values) : shapeResource.update(existing, values);

const ResourceForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
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
}: ResourceFormProps) => {
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
        fields: ["bio"],
        validationSchema: userTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
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

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<Resource>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Resource",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Resource, ResourceCreateInput, ResourceUpdateInput>({
        isCreate,
        isMutate,
        endpointCreate: endpointPostResource,
        endpointUpdate: endpointPutResource,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "Resource" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
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
                created_at: (existing as Resource)?.created_at ?? new Date().toISOString(),
                updated_at: (existing as Resource)?.updated_at ?? new Date().toISOString(),
            } as Resource);
        }
    }, [disabled, existing, fetch, handleCompleted, isCreate, isMutate, props, values]);

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
                title={isCreate ? t("CreateResource") : t("UpdateResource")}
                help={t("ResourceHelp")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {/* Enter link or search for object */}
                    <LinkInput onObjectData={foundLinkData} tabIndex={0} />
                    <Selector
                        name="usedFor"
                        options={Object.keys(ResourceUsedFor)}
                        getOptionIcon={(i) => getResourceIcon(i as ResourceUsedFor)}
                        getOptionLabel={(l) => t(l as CommonKey, { count: 2 })}
                        fullWidth
                        label={t("Type")}
                    />
                    <Divider />
                    <TranslatedTextInput
                        fullWidth
                        isOptional
                        label={t("Name")}
                        language={language}
                        name="name"
                    />
                    <TranslatedTextInput
                        fullWidth
                        isOptional
                        label={t("Description")}
                        language={language}
                        multiline
                        minRows={2}
                        maxRows={8}
                        name="description"
                    />
                    {/* Language select */}
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />
                </Stack>
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
        </MaybeLargeDialog>
    );
};

export const ResourceUpsert = ({
    isCreate,
    isMutate,
    isOpen,
    overrideObject,
    ...props
}: ResourceUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Resource, ResourceShape>({
        ...endpointGetResource,
        isCreate,
        objectType: "Resource",
        overrideObject: overrideObject as Resource,
        transform: (existing) => resourceInitialValues(session, existing as NewResourceShape),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformResourceValues, resourceValidation)}
        >
            {(formik) => <ResourceForm
                disabled={!(isCreate || canUpdate)}
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
};
