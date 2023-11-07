import { Box } from "@mui/material";
import { SearchType } from "utils/search/objectToSearch";
import { AdvancedSearchButton } from "../AdvancedSearchButton/AdvancedSearchButton";
import { SortButton } from "../SortButton/SortButton";
import { TimeButton } from "../TimeButton/TimeButton";
import { SearchButtonsProps } from "../types";

export const SearchButtons = ({
    advancedSearchParams,
    advancedSearchSchema,
    controlsUrl,
    searchType,
    setAdvancedSearchParams,
    setSortBy,
    setTimeFrame,
    sortBy,
    sortByOptions,
    sx,
    timeFrame,
}: SearchButtonsProps) => {
    return (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", marginBottom: 1, ...sx }}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
            />
            {searchType !== SearchType.Popular && <AdvancedSearchButton
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                controlsUrl={controlsUrl}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
            />}
        </Box>
    );
};
