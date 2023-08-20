/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { CommonKey, GqlModelType, VisibilityType } from "@local/shared";
import { ListItemIcon, ListItemText, Menu, MenuItem, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts/SessionContext";
import { useTabs } from "hooks/useTabs";
import { AddIcon, ApiIcon, HelpIcon, MonthIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SearchIcon, SmartContractIcon, StandardIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { MyStuffPageTabOption, SearchType } from "utils/search/objectToSearch";
import { MyStuffViewProps } from "../types";

// Data for each tab TODO add bot tab
const tabParams = [{
    Icon: RoutineIcon,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: MyStuffPageTabOption.Routine,
    where: () => ({ isInternal: false, visibility: VisibilityType.Own }),
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: MyStuffPageTabOption.Project,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: MonthIcon,
    titleKey: "Schedule" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: MyStuffPageTabOption.Schedule,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ReminderIcon,
    titleKey: "Reminder" as CommonKey,
    searchType: SearchType.Reminder,
    tabType: MyStuffPageTabOption.Reminder,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    searchType: SearchType.Note,
    tabType: MyStuffPageTabOption.Note,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    searchType: SearchType.Question,
    tabType: MyStuffPageTabOption.Question,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: MyStuffPageTabOption.Organization,
    where: (userId: string) => ({ memberUserIds: [userId] }),
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    searchType: SearchType.Standard,
    tabType: MyStuffPageTabOption.Standard,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    searchType: SearchType.Api,
    tabType: MyStuffPageTabOption.Api,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
    searchType: SearchType.SmartContract,
    tabType: MyStuffPageTabOption.SmartContract,
    where: () => ({ visibility: VisibilityType.Own }),
}];

export const MyStuffView = ({
    isOpen,
    onClose,
}: MyStuffViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const {
        id: userId,
        apisCount,
        membershipsCount,
        questionsAskedCount,
        smartContractsCount,
        standardsCount,
    } = useMemo(() => getCurrentUser(session), [session]);
    const display = toDisplay(isOpen);

    /**
     * Filter out certain tabs that we don't have any data for, 
     * so user isn't overwhelmed with tabs for objects they never worked with. 
     * Always keeps routines, projects, and notes
     */
    const filteredTabs = useMemo(() => tabParams.filter(tab => {
        switch (tab.tabType) {
            case MyStuffPageTabOption.Api:
                return Boolean(apisCount);
            case MyStuffPageTabOption.Organization:
                return Boolean(membershipsCount);
            case MyStuffPageTabOption.Question:
                return Boolean(questionsAskedCount);
            case MyStuffPageTabOption.SmartContract:
                return Boolean(smartContractsCount);
            case MyStuffPageTabOption.Standard:
                return Boolean(standardsCount);
        }
        return true;
    }), [apisCount, membershipsCount, questionsAskedCount, smartContractsCount, standardsCount]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MyStuffPageTabOption>({ tabParams: filteredTabs, display });

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === SearchType.Popular) {
            setSelectCreateTypeAnchorEl(e.currentTarget);
            return;
        }
        // Navigate to object's add page
        setLocation(`${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`);
    }, [searchType, setLocation]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType) => {
        if (type) setLocation(`${getObjectUrlBase({ __typename: type as `${GqlModelType}` })}/add`);
        else setSelectCreateTypeAnchorEl(null);
    }, [setLocation]);

    const focusSearch = useCallback(() => {
        const searchInput = document.getElementById("search-bar-my-stuff-list");
        searchInput?.focus();
    }, []);

    return (
        <>
            <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All' */}
                {tabParams.filter((t) => ![SearchType.Popular].includes(t.searchType)).map(tab => (
                    <MenuItem
                        key={tab.searchType}
                        onClick={() => onSelectCreateTypeClose(tab.searchType as SearchType)}
                    >
                        <ListItemIcon>
                            <tab.Icon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t(tab.searchType, { count: 1, defaultValue: tab.searchType })} />
                    </MenuItem>
                ))}
            </Menu>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("MyStuff")}
                below={<PageTabs
                    ariaLabel="my-stuff-tabs"
                    fullWidth
                    id="my-stuff-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="my-stuff-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                where={where(userId ?? "")}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionButtons
                display={display}
                sx={{ position: "fixed" }}
            >
                <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={focusSearch} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
                {userId ? (
                    <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={onCreateStart} >
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
