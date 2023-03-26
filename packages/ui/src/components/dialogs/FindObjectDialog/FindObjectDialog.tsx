import { Button, DialogContent, Stack, Typography } from "@mui/material";
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgProps, UserIcon, VisibleIcon } from "@shared/icons";
import { addSearchParams, parseSearchParams, useLocation } from "@shared/route";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AutocompleteOption } from "types";
import { getObjectUrl } from "utils/navigation/openObject";
import { SearchPageTabOption, SearchType } from "utils/search/objectToSearch";
import { FindObjectDialogProps } from "../types";

type BaseParams = {
    Icon: (props: SvgProps) => JSX.Element,
    searchType: 'All' | SearchType;
    tabType: 'All' | SearchPageTabOption;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    Icon: VisibleIcon,
    searchType: 'All',
    tabType: 'All',
    where: {},
}, {
    Icon: RoutineIcon,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routines,
    where: {},
}, {
    Icon: ProjectIcon,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Projects,
    where: {},
}, {
    Icon: HelpIcon,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Questions,
    where: {},
}, {
    Icon: NoteIcon,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Notes,
    where: {},
}, {
    Icon: OrganizationIcon,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organizations,
    where: {},
}, {
    Icon: UserIcon,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.Users,
    where: {},
}, {
    Icon: StandardIcon,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standards,
    where: {},
}, {
    Icon: ApiIcon,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Apis,
    where: {},
}, {
    Icon: SmartContractIcon,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContracts,
    where: {},
}];

const searchTitleId = "search-vrooli-for-link-title";

export const FindObjectDialog = ({
    handleClose,
    isOpen,
    zIndex,
}: FindObjectDialogProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Tabs to filter by object type
    const tabs = useMemo<PageTab<'All' | SearchPageTabOption>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<'All' | SearchPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to routine tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<SearchPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab)
    }, [setLocation]);

    const [searchString, setSearchString] = useState<string>('');
    const [selectedObject, setSelectedObject] = useState<{
        versions: { id: string; versionIndex: number, versionLabel: string }[];
    } | null>(null);
    // Reset search string and selected objectwhen dialog is opened/closed
    useEffect(() => {
        setSearchString('');
        setSelectedObject(null);
    }, [isOpen]);

    // On tab change, update BaseParams, document title, where, and URL
    const { searchType, where } = useMemo<BaseParams>(() => {
        return tabParams[currTab.index];
    }, [currTab.index]);

    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        console.log('onInputSelect', newValue)
        // If value is not an object, return;
        if (!newValue || newValue.__typename === 'Shortcut' || newValue.__typename === 'Action') return false;
        // If object has versions
        if ((newValue as any).versions && (newValue as any).versions.length > 0) {
            // If there is only one version, select it
            if ((newValue as any).versions.length === 1) {
                const objectUrl = getObjectUrl(newValue);
                // Select and close dialog
                handleClose(`${window.location.origin}${objectUrl}/${(newValue as any).versions[0].id}`);
                return false;
            }
            // Otherwise, set selected object so we can choose which version to link to
            setSelectedObject(newValue as any);
            return false;
        }
        // Otherweise, create URL
        const objectUrl = getObjectUrl(newValue);
        // Select and close dialog
        handleClose(`${window.location.origin}${objectUrl}`);
        return false;
    }, [handleClose]);

    const onVersionSelect = useCallback((version: { id: string }) => {
        if (!selectedObject) return;
        // Create base URL
        const objectUrl = getObjectUrl(selectedObject as any);
        // Select and close dialog
        handleClose(`${window.location.origin}${objectUrl}/${version.id}`);
    }, [handleClose, selectedObject]);

    return (
        <LargeDialog
            id="resource-find-object-dialog"
            isOpen={isOpen}
            onClose={handleClose}
            titleId={searchTitleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                titleData={{
                    hideOnDesktop: true,
                    titleKey: 'SearchVrooli',
                }}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <DialogContent sx={{
                overflowY: 'visible',
                minHeight: '500px',
            }}>
                {/* Search list to find object */}
                {!selectedObject && <SearchList
                    id="find-object-search-list"
                    beforeNavigation={onInputSelect}
                    take={20}
                    // Combine results for each object type
                    resolve={(data: { [x: string]: any }) => {
                        console.log('resolve 1', data);
                        // Find largest array length 
                        const max: number = Object.values(data).reduce((acc: number, val: any[]) => {
                            return Math.max(acc, val.length);
                        }, -Infinity);
                        console.log('resolve 2', max);
                        // Initialize result array
                        const result: any[] = [];
                        // Loop through each index
                        for (let i = 0; i < max; i++) {
                            // Loop through each object type
                            for (const key in data) {
                                // If index exists, push to result
                                if (Array.isArray(data[key]) && data[key][i]) result.push(data[key][i]);
                            }
                        }
                        console.log('resolve 3', result);
                        return result;
                    }}
                    searchType='Popular'
                    zIndex={zIndex}
                    where={{ ...where, objectType: searchType === 'All' ? undefined : searchType }}
                />}
                {/* If object selected (and supports versioning), display buttons to select version */}
                {selectedObject && (
                    <Stack spacing={2} direction="column" sx={{ mt: 2 }}>
                        <Typography variant="h6" mt={2} textAlign="center">
                            Select a version
                        </Typography>
                        {selectedObject.versions.sort((a, b) => b.versionIndex - a.versionIndex).map((version, index) => (
                            <Button
                                key={index}
                                fullWidth
                                onClick={() => onVersionSelect(version)}
                            >
                                {version.versionLabel}
                            </Button>
                        ))}
                        {/* Remove selection button */}
                        <Button
                            fullWidth
                            color="secondary"
                            onClick={() => setSelectedObject(null)}
                        >
                            Select a different object
                        </Button>
                    </Stack>
                )}
            </DialogContent>
        </LargeDialog>
    )
}