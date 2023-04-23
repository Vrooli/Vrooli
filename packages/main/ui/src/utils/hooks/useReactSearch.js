import { useCallback, useEffect, useState } from "react";
import { parseSearchParams } from "../route";
export const useReactSearch = (pollInterval = 50) => {
    const [search, setSearch] = useState(parseSearchParams());
    const listenToSearch = useCallback(() => {
        const newSearch = parseSearchParams();
        if (JSON.stringify(newSearch) !== JSON.stringify(search)) {
            setSearch(newSearch);
        }
    }, [search]);
    useEffect(() => {
        if (pollInterval) {
            const interval = setInterval(listenToSearch, 50);
            return () => clearInterval(interval);
        }
    }, [listenToSearch, pollInterval]);
    return search;
};
//# sourceMappingURL=useReactSearch.js.map