import { CommonKey, MemberInviteStatus } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useField } from "formik";
import { AddIcon, LockIcon, SearchIcon, UnlockIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { useTabs } from "utils/hooks/useTabs";
import { MemberManagePageTabOption, SearchType } from "utils/search/objectToSearch";
import { MemberManageViewProps } from "../types";

const tabParams = [{
    titleKey: "Member" as CommonKey,
    searchType: SearchType.Member,
    tabType: MemberManagePageTabOption.Member,
    where: (organizationId: string) => ({ organizationId }),
}, {
    titleKey: "Invite" as CommonKey,
    searchType: SearchType.MemberInvite,
    tabType: MemberManagePageTabOption.MemberInvite,
    where: (organizationId: string) => ({ organizationId, statuses: [MemberInviteStatus.Pending, MemberInviteStatus.Declined] }),
}];

/**
 * View members and invited members of an organization
 */
export const MemberManageView = ({
    onClose,
    organizationId,
    isOpen,
    zIndex,
}: MemberManageViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MemberManagePageTabOption>({ tabParams, display });

    const [isOpenToNewMembersField, , isOpenToNewMembersHelpers] = useField("isOpenToNewMembers");

    const [isInviteDialogOpen, setInviteDialogOpen] = useState(false);
    const onInviteStart = useCallback(() => {
        setInviteDialogOpen(true);
    }, []);

    const focusSearch = useCallback(() => {
        const searchInput = document.getElementById("search-bar-member-manage-list");
        searchInput?.focus();
    }, []);

    return (
        <MaybeLargeDialog
            display={display}
            id="member-manage-dialog"
            isOpen={isOpen ?? false}
            onClose={onClose}
            zIndex={zIndex}
            sxs={{
                paper: {
                    minHeight: "min(100vh - 64px, 600px)",
                },
            }}
        >
            {/* Dialog for creating new member invite TODO */}
            {/* <MemberInviteUpsert
                    display="dialog"
                    isCreate={true}
                    isOpen={isInviteDialogOpen}
                    onCompleted={handleCreated}
                    onCancel={handleCreateClose}
                    zIndex={zIndex + 2}
                /> */}
            {/* Main dialog */}
            <TopBar
                display={display}
                onClose={onClose}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    fullWidth
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
                        onClick={onInviteStart}
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
                id="member-manage-list"
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                zIndex={zIndex}
                where={where(organizationId)}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={focusSearch} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
                <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={onInviteStart} >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
            </SideActionButtons>
        </MaybeLargeDialog>
    );
};
