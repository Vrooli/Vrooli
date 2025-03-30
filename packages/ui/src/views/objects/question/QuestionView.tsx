import { CommentFor, endpointsQuestion, exists, Question, Tag, TagShape } from "@local/shared";
import { IconButton, Stack, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { DateDisplay } from "../../../components/text/DateDisplay.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { FormSection } from "../../../styles.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { questionInitialValues } from "./QuestionUpsert.js";
import { QuestionViewProps } from "./types.js";

const contextActionsExcluded = [ObjectAction.Edit] as const;

export function QuestionView({
    display,
    onClose,
}: QuestionViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setQuestion } = useManagedObject<Question>({
        ...endpointsQuestion.findOne,
        objectType: "Question",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { title, subtitle } = useMemo(() => getDisplay(existing, [language]), [existing, language]);

    const initialValues = useMemo(() => questionInitialValues(session, existing), [existing, session]);
    const tags = useMemo<TagShape[] | null | undefined>(() => initialValues?.tags as TagShape[] | null | undefined, [initialValues]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Question",
        openAddCommentDialog,
        setLocation,
        setObject: setQuestion,
    });

    const comments = useMemo(() => (
        <FormSection>
            <CommentContainer
                forceAddCommentOpen={isAddCommentOpen}
                language={language}
                objectId={existing?.id ?? ""}
                objectType={CommentFor.Question}
                onAddCommentClose={closeAddCommentDialog}
            />
        </FormSection>
    ), [closeAddCommentDialog, existing?.id, isAddCommentOpen, language]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(title, t("Question", { count: 1 }))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                />}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={() => { }}
            >
                {() => <Stack direction="column" spacing={4} sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 700px)",
                    padding: 2,
                }}>
                    <RelationshipList
                        isEditing={false}
                        objectType={"Question"}
                    />
                    {/* Date and tags */}
                    <Stack direction="row" spacing={1} mt={2} mb={1}>
                        {/* Date created */}
                        <DateDisplay
                            loading={isLoading}
                            showIcon={true}
                            timestamp={existing?.created_at}
                        />
                        {exists(tags) && tags.length > 0 && <TagList
                            maxCharacters={30}
                            parentId={existing?.id ?? ""}
                            tags={tags as Tag[]}
                            sx={{ marginTop: 4 }}
                        />}
                    </Stack>
                    <MarkdownDisplay content={subtitle} />
                    {/* Action buttons */}
                    <ObjectActionsRow
                        actionData={actionData}
                        exclude={contextActionsExcluded} // Handled elsewhere
                        object={existing}
                    />
                    {/* Comments */}
                    {comments}
                </Stack>}
            </Formik>
            <SideActionsButtons display={display}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton
                        aria-label={t("UpdateQuestion")}
                        onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}
                    >
                        <IconCommon name="Edit" />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
