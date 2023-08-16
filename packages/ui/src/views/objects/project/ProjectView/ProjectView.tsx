import { BookmarkFor, endpointGetProjectVersion, LINKS, ProjectVersion } from "@local/shared";
import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { EditIcon, EllipsisIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { OverviewContainer } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { toDisplay } from "utils/display/pageTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ProjectViewProps } from "../types";

export const ProjectView = ({
    isOpen,
    onClose,
    zIndex,
}: ProjectViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading, object: existing, permissions, setObject: setProjectVersion } = useObjectFromUrl<ProjectVersion>({
        ...endpointGetProjectVersion,
        objectType: "ProjectVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, handle } = useMemo(() => {
        const { description, name } = getTranslation(existing, [language]);
        return {
            name,
            description,
            handle: existing?.root?.handle,
        };
    }, [language, existing]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "ProjectVersion",
        setLocation,
        setObject: setProjectVersion,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                tabTitle={handle ? `${name} (@${handle})` : name}
                zIndex={zIndex}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={existing as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <OverviewContainer>
                <Stack direction="row" mr={2}>
                    <Tooltip title={t("MoreOptions")}>
                        <IconButton
                            aria-label={t("MoreOptions")}
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
                    <BookmarkButton
                        objectId={existing?.id ?? ""}
                        bookmarkFor={BookmarkFor.Project}
                        isBookmarked={existing?.root?.you?.isBookmarked ?? false}
                        bookmarks={existing?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </Stack>
                <Stack direction="column" spacing={1} p={2} justifyContent="center" sx={{
                    alignItems: "flex-start",
                }}>
                    {/* Title */}
                    {
                        (isLoading && !name) ? (
                            <TextLoading size="header" sx={{ width: "50%" }} />
                        ) : <Title
                            title={name}
                            variant="header"
                            options={permissions.canUpdate ? [{
                                label: t("Edit"),
                                Icon: EditIcon,
                                onClick: () => { actionData.onActionStart("Edit"); },
                            }] : []}
                            zIndex={zIndex}
                            sxs={{ stack: { padding: 0, paddingBottom: handle ? 0 : 2 } }}
                        />
                    }
                    {/* Handle */}
                    {
                        handle && <Typography
                            variant="h6"
                            textAlign="center"
                            fontFamily="monospace"
                            onClick={() => {
                                navigator.clipboard.writeText(`${window.location.origin}${LINKS.Project}/${handle}`);
                                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
                            }}
                            sx={{
                                color: palette.secondary.dark,
                                cursor: "pointer",
                                paddingBottom: 2,
                            }}
                        >@{handle}</Typography>
                    }
                    {/* Description */}
                    {
                        (isLoading && !description) ? (
                            <TextLoading lines={2} size="body1" sx={{ width: "85%" }} />
                        ) : (
                            <MarkdownDisplay
                                variant="body1"
                                sx={{ color: description ? palette.background.textPrimary : palette.background.textSecondary }}
                                content={description ?? "No description set"}
                                zIndex={zIndex}
                            />
                        )
                    }
                    <Stack direction="row" spacing={2} sx={{
                        alignItems: "center",
                    }}>
                        {/* Created date */}
                        <DateDisplay
                            loading={isLoading}
                            showIcon={true}
                            textBeforeDate="Created"
                            timestamp={existing?.created_at}
                            zIndex={zIndex}
                        />
                        <ReportsLink object={existing} />
                    </Stack>
                </Stack>
            </OverviewContainer>
            {/* View routines and standards associated with this project */}
            <Box>
                {/* Breadcrumbs to show directory hierarchy */}
                {/* TODO */}
                {/* List of items in current directory */}
                {/* <SearchList
                    canSearch={() => Boolean(projectVersion?.id)}
                    dummyLength={display === "page" ? 5 : 3}
                    handleAdd={permissions.canUpdate ? toAddNew : undefined}
                    hideUpdateButton={true}
                    id="directory-view-list"
                    searchType={searchType}
                    searchPlaceholder={placeholder}
                    take={20}
                    where={where}
                    zIndex={zIndex}
                /> */}
            </Box>
            {/* Edit button (if canUpdate) */}
            <SideActionButtons
                // Treat as a dialog when build view is open
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {permissions.canUpdate ? (
                    <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
