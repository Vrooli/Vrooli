import { CommentFor, EditIcon, endpointGetQuestion, exists, Question, Tag, useLocation } from "@local/shared";
import { Box, Stack, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { CommentContainer, containerProps } from "components/containers/CommentContainer/CommentContainer";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Formik } from "formik";
import { questionInitialValues } from "forms/QuestionForm/QuestionForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ObjectAction } from "utils/actions/objectActions";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { TagShape } from "utils/shape/models/tag";
import { QuestionViewProps } from "../types";

export const QuestionView = ({
    display = "page",
    onClose,
    partialData,
    zIndex,
}: QuestionViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setQuestion } = useObjectFromUrl<Question>({
        ...endpointGetQuestion,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { title, subtitle } = useMemo(() => getDisplay(existing, [language]), [existing, language]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

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
        <Box sx={containerProps(palette)}>
            <CommentContainer
                forceAddCommentOpen={isAddCommentOpen}
                language={language}
                objectId={existing?.id ?? ""}
                objectType={CommentFor.Question}
                onAddCommentClose={closeAddCommentDialog}
                zIndex={zIndex}
            />
        </Box>
    ), [closeAddCommentDialog, existing?.id, isAddCommentOpen, language, palette, zIndex]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(title, t("Question"))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                    zIndex={zIndex}
                />}
                zIndex={zIndex}
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
                        zIndex={zIndex}
                    />
                    {/* Date and tags */}
                    <Stack direction="row" spacing={1} mt={2} mb={1}>
                        {/* Date created */}
                        <DateDisplay
                            loading={isLoading}
                            showIcon={true}
                            timestamp={existing?.created_at}
                            zIndex={zIndex}
                        />
                        {exists(tags) && tags.length > 0 && <TagList
                            maxCharacters={30}
                            parentId={existing?.id ?? ""}
                            tags={tags as Tag[]}
                            sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
                        />}
                    </Stack>
                    <MarkdownDisplay content={subtitle} zIndex={zIndex} />
                    {/* Action buttons */}
                    <ObjectActionsRow
                        actionData={actionData}
                        exclude={[ObjectAction.Edit]} // Handled elsewhere
                        object={existing}
                        zIndex={zIndex}
                    />
                    {/* Comments */}
                    {comments}
                </Stack>}
            </Formik>
            {/* Edit button (if canUpdate)*/}
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
