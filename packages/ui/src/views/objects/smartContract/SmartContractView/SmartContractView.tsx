import { BookmarkFor, EditIcon, EllipsisIcon, endpointGetSmartContractVersion, SmartContractIcon, SmartContractVersion, useLocation } from "@local/shared";
import { Avatar, Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { Title } from "components/text/Title/Title";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { OverviewContainer } from "styles";
import { placeholderColor } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { SmartContractViewProps } from "../types";

export const SmartContractView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: SmartContractViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: smartContractVersion, permissions, setObject: setSmartContractVersion } = useObjectFromUrl<SmartContractVersion>({
        ...endpointGetSmartContractVersion,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (smartContractVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [smartContractVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(smartContractVersion ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
        };
    }, [language, smartContractVersion, partialData]);

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
        object: smartContractVersion,
        objectType: "SmartContractVersion",
        setLocation,
        setObject: setSmartContractVersion,
    });

    /**
     * Displays name, avatar, description, and quick links
     */
    const overviewComponent = useMemo(() => (
        <OverviewContainer>
            <Avatar
                src="/broken-image.jpg" //TODO
                sx={{
                    backgroundColor: profileColors[0],
                    color: profileColors[1],
                    boxShadow: 2,
                    transform: "translateX(-50%)",
                    width: "min(100px, 25vw)",
                    height: "min(100px, 25vw)",
                    left: "50%",
                    top: "-55px",
                    position: "absolute",
                    fontSize: "min(50px, 10vw)",
                }}
            >
                <SmartContractIcon fill="white" width='75%' height='75%' />
            </Avatar>
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
                    ) : <Title
                        title={name}
                        variant="header"
                        options={permissions.canUpdate ? [{
                            label: t("Edit"),
                            Icon: EditIcon,
                            onClick: () => { actionData.onActionStart("Edit"); },
                        }] : []}
                    />
                }
                {/* Joined date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={smartContractVersion?.created_at}
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
                    <ShareButton object={smartContractVersion} zIndex={zIndex} />
                    <ReportsLink object={smartContractVersion} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        objectId={smartContractVersion?.id ?? ""}
                        bookmarkFor={BookmarkFor.SmartContract}
                        isBookmarked={smartContractVersion?.root?.you?.isBookmarked ?? false}
                        bookmarks={smartContractVersion?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </Stack>
            </Stack>
        </OverviewContainer>
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, profileColors, openMoreMenu, isLoading, name, permissions.canUpdate, permissions.canBookmark, t, smartContractVersion, description, zIndex, actionData]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("SmartContract")}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={smartContractVersion as any}
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
                    {availableLanguages.length > 1 && <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                        zIndex={zIndex}
                    />}
                </Box>
                {overviewComponent}
            </Box>
            {/* TODO */}
        </>
    );
};
