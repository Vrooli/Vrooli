import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box } from "@mui/material";
import { AdvancedSearchButton } from "../AdvancedSearchButton/AdvancedSearchButton";
import { SortButton } from "../SortButton/SortButton";
import { TimeButton } from "../TimeButton/TimeButton";
export const SearchButtons = ({ advancedSearchParams, advancedSearchSchema, searchType, setAdvancedSearchParams, setSortBy, setTimeFrame, sortBy, sortByOptions, timeFrame, zIndex, }) => {
    return (_jsxs(Box, { sx: { display: "flex", justifyContent: "center", alignItems: "center", marginBottom: 1 }, children: [_jsx(SortButton, { options: sortByOptions, setSortBy: setSortBy, sortBy: sortBy }), _jsx(TimeButton, { setTimeFrame: setTimeFrame, timeFrame: timeFrame }), _jsx(AdvancedSearchButton, { advancedSearchParams: advancedSearchParams, advancedSearchSchema: advancedSearchSchema, searchType: searchType, setAdvancedSearchParams: setAdvancedSearchParams, zIndex: zIndex })] }));
};
//# sourceMappingURL=SearchButtons.js.map