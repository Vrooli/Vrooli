import { Box, Button, Tooltip, Typography } from "@mui/material";
import { SearchBreadcrumbs, SearchList } from "components";
import { CSSProperties, useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { BaseSearchPageProps } from "./types";
import { SearchQueryVariablesInput } from "components/lists/types";

export function BaseSearchPage<DataType, SortBy>({
    title = 'Search',
    searchPlaceholder = 'Search...',
    sortOptions,
    defaultSortOption,
    query,
    take = 20,
    listItemFactory,
    getOptionLabel,
    onObjectSelect,
    popupButtonText,
    popupButtonTooltip = "Couldn't find what you were looking for? Try creating your own!",
    onPopupButtonClick,
}: BaseSearchPageProps<DataType, SortBy>) {
    const [addNewButton, setAddNewButton] = useState<boolean>(false);
    const handleScrolledFar = useCallback(() => { setAddNewButton(true) }, [])
    const addNewButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={popupButtonTooltip}>
                <Button
                    onClick={onPopupButtonClick}
                    size="large"
                    sx={{
                        zIndex: 100,
                        minidth: 'min(100%, 200px)',
                        height: '48px',
                        borderRadius: 3,
                        position: 'fixed',
                        bottom: '5em',
                        transform: addNewButton ? 'translateY(0)' : 'translateY(10em)',
                        transition: 'transform 1s ease-in-out',

                    } as CSSProperties}
                >
                    {popupButtonText}
                </Button>
            </Tooltip>
        </Box>
    ), [addNewButton, onPopupButtonClick, popupButtonText, popupButtonTooltip]);

    return (
        <Box id="page">
            <SearchBreadcrumbs sx={{ ...centeredDiv, color: (t) => t.palette.secondary.dark }} />
            <Typography component="h2" variant="h4" textAlign="center" sx={{ paddingTop: 2 }}>{title}</Typography>
            <SearchList
                searchPlaceholder={searchPlaceholder}
                sortOptions={sortOptions}
                defaultSortOption={defaultSortOption}
                query={query}
                take={take}
                listItemFactory={listItemFactory}
                getOptionLabel={getOptionLabel}
                onObjectSelect={onObjectSelect}
                onScrolledFar={handleScrolledFar}
            />
            {addNewButtonContainer}
        </Box>
    )
}