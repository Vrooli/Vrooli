import { DUMMY_ID, NoteVersion, noteVersionTranslationValidation, noteVersionValidation, orDefault, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { NoteFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NoteVersionShape, shapeNoteVersion } from "utils/shape/models/noteVersion";
import { OwnerShape } from "utils/shape/models/types";

export const noteInitialValues = (
    session: Session | undefined,
    existing?: Partial<NoteVersion> | null | undefined,
): NoteVersionShape => ({
    __typename: "NoteVersion" as const,
    id: DUMMY_ID,
    directoryListings: [],
    isPrivate: true,
    versionLabel: existing?.versionLabel ?? "1.0.0",
    ...existing,
    root: {
        id: DUMMY_ID,
        isPrivate: true,
        owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" } as OwnerShape,
        parent: null,
        tags: [],
        ...existing?.root,
    },
    translations: orDefault(existing?.translations, [{
        __typename: "NoteVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "New Note",
        pages: [{
            __typename: "NotePage" as const,
            text: "",
        }],
    }]),
});

export const transformNoteValues = (values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) =>
    isCreate ? shapeNoteVersion.create(values) : shapeNoteVersion.update(existing, values);

export const validateNoteValues = async (values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) => {
    const transformedValues = transformNoteValues(values, existing, isCreate);
    const validationSchema = noteVersionValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const NoteForm = forwardRef<BaseFormRef | undefined, NoteFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    onClose,
    values,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name", "text"],
        validationSchema: noteVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    console.log("errors", combineErrorsWithTranslations(props.errors, translationErrors));

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title=""
                titleComponent={<EditableTitle
                    language={language}
                    titleField="name"
                    subtitleField="description"
                    validationEnabled={false}
                    variant="subheader"
                    sxs={{ stack: { padding: 0 } }}
                />}
            />
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                ref={ref}
                style={{
                    width: "min(800px, 100vw)",
                    paddingBottom: 0,
                }}
            >
                <TranslatedMarkdownInput
                    language={language}
                    name="text"
                    placeholder={t("PleaseBeNice")}
                    minRows={10}
                    sxs={{
                        bar: {
                            borderRadius: 0,
                            background: palette.primary.main,
                            position: "sticky",
                            top: 0,
                        },
                        root: {
                            height: "100%",
                            position: "relative",
                            maxWidth: "800px",
                            ...(display === "page" ? {
                                marginBottom: 4,
                                borderRadius: { xs: 0, md: 1 },
                                overflow: "overlay",
                            } : {}),
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: "none",
                            height: "100%",
                            overflow: "hidden", // Container handles scrolling
                            background: palette.background.paper,
                            border: "none",
                            ...(display === "page" ? {
                                minHeight: "100vh",
                            } : {}),
                        },
                    }}
                />
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                sideActionButtons={{
                    display,
                    children: (
                        <EllipsisActionButton>
                            <RelationshipList
                                isEditing={true}
                                objectType={"Note"}
                            />
                        </EllipsisActionButton>
                    ),
                }}
            />
        </>
    );
});
