import { BookmarkFor, EditIcon, EllipsisIcon, FindByIdInput, HelpIcon, Question } from "@local/shared";
import { Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { questionFindOne } from "../../../api/generated/endpoints/question_findOne";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "../../../components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { SessionContext } from "../../../utils/SessionContext";
import { placeholderColor } from "../../../utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { QuestionViewProps } from "../types";

export const QuestionView = ({
    display = "page",
    partialData,
    zIndex = 200,
}: QuestionViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: question, permissions, setObject: setQuestion } = useObjectFromUrl<Question, FindByIdInput>({
        query: questionFindOne,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (question?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [question?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(question ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
        };
    }, [language, question, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: question,
        objectType: "Question",
        setLocation,
        setObject: setQuestion,
    });

    /**
     * Displays name, avatar, description, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            position="relative"
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: "0", sm: 2 },
                boxShadow: { xs: "none", sm: 12 },
                width: { xs: "100%", sm: "min(500px, 100vw)" },
            }}
        >
            <Box
                width={"min(100px, 25vw)"}
                height={"min(100px, 25vw)"}
                borderRadius='100%'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top="-55px"
                sx={{
                    border: "1px solid black",
                    backgroundColor: profileColors[0],
                    transform: "translateX(-50%)",
                }}
            >
                <HelpIcon fill={profileColors[1]} width='80%' height='80%' />
            </Box>
            <Tooltip title="See all options">
                <IconButton
                    aria-label="More"
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : permissions.canUpdate ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit question">
                                <IconButton
                                    aria-label="Edit question"
                                    size="small"
                                    onClick={() => actionData.onActionStart("Edit")}
                                >
                                    <EditIcon fill={palette.secondary.main} />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{name}</Typography>
                    )
                }
                {/* Joined date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={question?.created_at}
                    width={"33%"}
                />
                {/* Bio */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "85%", color: "grey.500" }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: description ? palette.background.textPrimary : palette.background.textSecondary }}>{description ?? "No description set"}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <ShareButton object={question} zIndex={zIndex} />
                    <ReportsLink object={question} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        objectId={question?.id ?? ""}
                        bookmarkFor={BookmarkFor.Question}
                        isBookmarked={question?.you?.isBookmarked ?? false}
                        bookmarks={question?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, question, description, zIndex, actionData]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "Question",
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={question as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                display: "flex",
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: "absolute",
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
            {/* TODO */}
        </>
    );
};
