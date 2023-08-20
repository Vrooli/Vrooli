import { commentTranslationValidation } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { CommentDialogProps } from "../types";

const titleId = "comment-dialog-title";

/**
 * Dialog for creating/updating a comment. 
 * Only used on mobile; desktop displays MarkdownInput at top of 
 * CommentContainer
 */
export const CommentDialog = ({
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    parent,
    ref,
    zIndex,
    ...props
}: CommentDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["text"],
        validationSchema: commentTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    const { subtitle: parentText } = useMemo(() => getDisplay(parent, [language]), [language, parent]);

    return (
        <LargeDialog
            id="comment-dialog"
            isOpen={isOpen}
            onClose={onCancel}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={onCancel}
                title={t(isCreate ? "AddComment" : "EditComment")}
                titleId={titleId}
                zIndex={zIndex}
            />
            <BaseForm
                dirty={dirty}
                display="dialog"
                isLoading={isLoading}
                ref={ref}
                style={{
                    width: "min(700px, 100vw)",
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
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: "none",
                            height: "100%",
                            overflow: "hidden", // Container handles scrolling
                            background: palette.background.paper,
                            border: "none",
                        },
                    }}
                    zIndex={zIndex}
                />
                {/* Display parent underneath */}
                {parent && (
                    <Box sx={{
                        backgroundColor: palette.background.paper,
                        height: "30vh",
                    }}>
                        <Typography variant="body2">{parentText}</Typography>
                    </Box>
                )}
                <GridSubmitButtons
                    display={"dialog"}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                    zIndex={zIndex}
                />
            </BaseForm>
        </LargeDialog>
    );
};
