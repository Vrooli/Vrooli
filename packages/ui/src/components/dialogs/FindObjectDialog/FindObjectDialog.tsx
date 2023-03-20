import { useQuery } from "@apollo/client";
import { Button, DialogContent, Stack, Typography, useTheme } from "@mui/material";
import { PopularInput, PopularResult } from "@shared/consts";
import { feedPopular } from "api/generated/endpoints/feed_popular";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "components/inputs/search";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AutocompleteOption, Wrap } from "types";
import { listToAutocomplete } from "utils/display/listTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { FindObjectDialogProps } from "../types";

const searchTitleId = "search-vrooli-for-link-title"

export const FindObjectDialog = ({
    handleClose,
    isOpen,
    zIndex,
}: FindObjectDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [searchString, setSearchString] = useState<string>('');
    const [selectedObject, setSelectedObject] = useState<{
        versions: { id: string; versionIndex: number, versionLabel: string }[];
    } | null>(null);
    // Reset search string and selected objectwhen dialog is opened/closed
    useEffect(() => {
        setSearchString('');
        setSelectedObject(null);
    }, [isOpen]);

    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const { data: searchData, refetch: refetchSearch, loading: searchLoading } = useQuery<Wrap<PopularResult, 'popular'>, Wrap<PopularInput, 'input'>>(feedPopular, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { isOpen && refetchSearch() }, [isOpen, refetchSearch, searchString]);
    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        console.log('calculating autocomplete options', searchData)
        const firstResults: AutocompleteOption[] = [];
        // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
        const flattened = (Object.values(searchData?.popular ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, ['en']).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems];
    }, [searchData]);

    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        console.log('onInputSelect', newValue)
        // If value is not an object, return;
        if (!newValue || newValue.__typename === 'Shortcut' || newValue.__typename === 'Action') return;
        // If object has versions (and more than one), set selected object so we can choose which version to link to
        if ((newValue as any).versions && (newValue as any).versions.length > 1) {
            setSelectedObject(newValue as any);
            return;
        }
        // Otherweise, create URL
        const objectUrl = getObjectUrl(newValue);
        // Select and close dialog
        handleClose(`${window.location.origin}${objectUrl}`);
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
            <DialogTitle
                id={searchTitleId}
                title={t('SearchVrooli')}
                onClose={handleClose}
            />
            <DialogContent sx={{
                overflowY: 'visible',
                minHeight: '500px',
            }}>
                {/* Search bar to find object */}
                <SiteSearchBar
                    id="vrooli-object-search"
                    autoFocus={true}
                    placeholder='SearchObjectLink'
                    options={autocompleteOptions}
                    loading={searchLoading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    showSecondaryLabel={true}
                    sxs={{
                        root: { width: '100%', top: 0, marginTop: 2 },
                        paper: { background: palette.background.paper },
                    }}
                />
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
                    </Stack>
                )}
            </DialogContent>
        </LargeDialog>
    )
}