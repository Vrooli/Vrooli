import { AddIcon, CommonKey, HistoryIcon, LockIcon, MemberInviteStatus, UnlockIcon, UserIcon } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { centeredDiv } from "styles";
import { useTabs } from "utils/hooks/useTabs";
import { MemberManagePageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { MemberManageViewProps } from "../types";

/**
 * View members and invited members of an organization
 */
export const MemberManageView = ({
    display = "dialog",
    onClose,
    organizationId,
    zIndex,
}: MemberManageViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => setShareDialogOpen(false), []);

    // Data for each tab
    const tabParams = useMemo(() => ([{
        Icon: UserIcon,
        titleKey: "Member" as CommonKey,
        searchType: SearchType.Member,
        tabType: MemberManagePageTabOption.Members,
        where: { organizationId },
    }, {
        Icon: HistoryIcon,
        titleKey: "Invite" as CommonKey,
        searchType: SearchType.MemberInvite,
        tabType: MemberManagePageTabOption.MemberInvites,
        where: { statuses: [MemberInviteStatus.Pending, MemberInviteStatus.Declined] },
    }] as const), [organizationId]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MemberManagePageTabOption>(tabParams, 0);

    const [isOpenToNewMembersField, , isOpenToNewMembersHelpers] = useField("isOpenToNewMembers");

    const [isInviteDialogOpen, setInviteDialogOpen] = useState(false);
    const onInviteClick = useCallback(() => {
        setInviteDialogOpen(true);
    }, [setInviteDialogOpen]);

    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={"Invite a user to join your organization"}>
                <Button
                    onClick={onInviteClick}
                    size="large"
                    variant="contained"
                    sx={{
                        zIndex: 100,
                        minWidth: "min(100%, 200px)",
                        height: "48px",
                        borderRadius: 3,
                        position: "fixed",
                        bottom: "calc(5em + env(safe-area-inset-bottom))",
                        transform: popupButton ? "translateY(0)" : "translateY(calc(10em + env(safe-area-inset-bottom)))",
                        transition: "transform 1s ease-in-out",
                    }}
                >
                    {t("Invite")}
                </Button>
            </Tooltip>
        </Box>
    ), [onInviteClick, popupButton, t]);

    return (
        <>
            {/* Dialog for creating new member invite TODO */}
            {/* <LargeDialog
                id="invite-member-dialog"
                onClose={handleCreateClose}
                isOpen={isInviteDialogOpen}
                titleId="invite-member-dialog-title"
                zIndex={zIndex + 2}
            >
                <MemberInviteUpsert
                    display="dialog"
                    isCreate={true}
                    onCompleted={handleCreated}
                    onCancel={handleCreateClose}
                    zIndex={zIndex + 1002}
                />
            </LargeDialog> */}
            {/* Main dialog */}
            <TopBar
                display={display}
                onClose={onClose}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(searchType as CommonKey, { count: 2, defaultValue: searchType })}</Typography>
                <Tooltip title={t("Invite")} placement="top">
                    <IconButton
                        size="medium"
                        onClick={onInviteClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                    </IconButton>
                </Tooltip>
                <Tooltip title="Indicates if this organization should be displayed when users are looking for an organization to join" placement="top">
                    <IconButton
                        size="medium"
                        onClick={() => isOpenToNewMembersHelpers.setValue(!isOpenToNewMembersField.value)}
                        sx={{
                            padding: 1,
                        }}
                    >
                        {isOpenToNewMembersField.value === true ? <UnlockIcon fill={palette.secondary.main} width='1.5em' height='1.5em' /> : <LockIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />}
                    </IconButton>
                </Tooltip>
            </Stack >
            {searchType && <SearchList
                id="main-search-page-list"
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                onScrolledFar={handleScrolledFar}
                zIndex={zIndex}
                where={where}
            />}
            {popupButtonContainer}
        </>
    );
};
