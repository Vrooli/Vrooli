import { Box, Typography, useTheme } from "@mui/material";
import { commentTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { CommentDialogProps } from "../types";

const titleId = 'comment-dialog-title';

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
        fields: ['text'],
        validationSchema: commentTranslationValidation[isCreate ? 'create' : 'update']({}),
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
                titleData={{ titleId, titleKey: isCreate ? 'AddComment' : 'EditComment' }}
            />
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    paddingBottom: '64px',
                }}
            >
                <TranslatedMarkdownInput
                    language={language}
                    name="text"
                    placeholder={t(`PleaseBeNice`)}
                    minRows={3}
                    sxs={{
                        bar: {
                            borderRadius: 0,
                            background: palette.primary.main,
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: 'none',
                            height: parent ? '70vh' : '100vh',
                            background: palette.background.paper,
                        }
                    }}
                />
                {/* Display parent underneath */}
                {parent && (
                    <Box sx={{
                        backgroundColor: palette.background.paper,
                        height: '30vh',
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
                />
            </BaseForm>
        </LargeDialog>
    )
}