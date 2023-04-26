import { DUMMY_ID, NoteVersion, noteVersionTranslationValidation, noteVersionValidation, orDefault, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { NoteFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NoteVersionShape, shapeNoteVersion } from "utils/shape/models/noteVersion";
import { OwnerShape } from "utils/shape/models/types";

export const noteInitialValues = (
    session: Session | undefined,
    existing?: NoteVersion | null | undefined,
): NoteVersionShape => ({
    __typename: "NoteVersion" as const,
    id: DUMMY_ID,
    directoryListings: [],
    isPrivate: true,
    root: {
        id: DUMMY_ID,
        isPrivate: true,
        owner: { __typename: "User", id: getCurrentUser(session)?.id! } as OwnerShape,
        parent: null,
        tags: [],
    },
    versionLabel: existing?.versionLabel ?? "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "NoteVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
        text: "",
    }]),
});

export function transformNoteValues(values: NoteVersionShape, existing?: NoteVersionShape) {
    return existing === undefined
        ? shapeNoteVersion.create(values)
        : shapeNoteVersion.update(existing, values);
}

export const validateNoteValues = async (values: NoteVersionShape, existing?: NoteVersionShape) => {
    const transformedValues = transformNoteValues(values, existing);
    const validationSchema = noteVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const NoteForm = forwardRef<any, NoteFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
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

    return (
        <>
            <SideActionButtons display={display} hasGridActions={true} zIndex={zIndex + 1}>
                <EllipsisActionButton>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Note"}
                        zIndex={zIndex}
                    />
                </EllipsisActionButton>
            </SideActionButtons>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(100vw - 16px, 700px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <TranslatedMarkdownInput
                    language={language}
                    name="text"
                    placeholder={t("PleaseBeNice")}
                    minRows={3}
                    sxs={{
                        bar: {
                            borderRadius: 0,
                            background: palette.primary.main,
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: "none",
                            minHeight: "100vh",
                            background: palette.background.paper,
                        },
                    }}
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
    );
});
