import { Box } from "@mui/material";
import { AdvancedSearchButton } from "../AdvancedSearchButton/AdvancedSearchButton";
import { SortButton } from "../SortButton/SortButton";
import { TimeButton } from "../TimeButton/TimeButton";
import { SearchButtonsProps } from "../types";

export const SearchButtons = ({
    advancedSearchParams,
    advancedSearchSchema,
    searchType,
    setAdvancedSearchParams,
    setSortBy,
    setTimeFrame,
    sortBy,
    sortByOptions,
    timeFrame,
    zIndex,
}: SearchButtonsProps) => {
    return (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", marginBottom: 1 }}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
                zIndex={zIndex}
            />
            <AdvancedSearchButton
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
                zIndex={zIndex}
            />
        </Box>
    );
};
