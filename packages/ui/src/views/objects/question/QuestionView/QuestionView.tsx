import { CommentFor, endpointGetQuestion, exists, Question, Tag, TagShape } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection, SideActionsButton } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { questionInitialValues } from "../QuestionUpsert/QuestionUpsert";
import { QuestionViewProps } from "../types";

export function QuestionView({
    display,
    onClose,
}: QuestionViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setQuestion } = useObjectFromUrl<Question>({
        ...endpointGetQuestion,
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
    ), [closeAddCommentDialog, existing?.id, isAddCommentOpen, language, palette]);

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
                        exclude={[ObjectAction.Edit]} // Handled elsewhere
                        object={existing}
                    />
                    {/* Comments */}
                    {comments}
                </Stack>}
            </Formik>
            <SideActionsButtons display={display}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <SideActionsButton aria-label={t("UpdateQuestion")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
