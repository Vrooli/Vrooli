import { exists } from "@local/utils";
import { useEffect, useMemo, useState } from "react";
import { useCustomLazyQuery } from "../../api";
import { defaultYou, getYou } from "../display/listTools";
import { parseSingleItemUrl } from "../navigation/urlTools";
import { PubSub } from "../pubsub";
import { useDisplayApolloError } from "./useDisplayApolloError";
import { useStableObject } from "./useStableObject";
export function useObjectFromUrl({ query, onInvalidUrlParams, partialData, idFallback, }) {
    const urlParams = useMemo(() => parseSingleItemUrl(), []);
    const stableOnInvalidUrlParams = useStableObject(onInvalidUrlParams);
    const [getData, { data, error, loading: isLoading }] = useCustomLazyQuery(query, { errorPolicy: "all" });
    const [object, setObject] = useState(null);
    useDisplayApolloError(error);
    useEffect(() => {
        console.log("parseSingleItemUrl", urlParams);
        if (exists(urlParams.handle))
            getData({ variables: { handle: urlParams.handle } });
        else if (exists(urlParams.handleRoot))
            getData({ variables: { handleRoot: urlParams.handleRoot } });
        else if (exists(urlParams.id))
            getData({ variables: { id: urlParams.id } });
        else if (exists(urlParams.idRoot))
            getData({ variables: { idRoot: urlParams.idRoot } });
        else if (exists(idFallback))
            getData({ variables: { id: idFallback } });
        else if (exists(stableOnInvalidUrlParams))
            stableOnInvalidUrlParams(urlParams);
        else
            PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, idFallback, stableOnInvalidUrlParams, urlParams]);
    useEffect(() => {
        setObject(data ?? partialData);
    }, [data, partialData]);
    const permissions = useMemo(() => object ? getYou(object) : defaultYou, [object]);
    return {
        id: object?.id ?? urlParams.id,
        isLoading,
        object,
        permissions,
        setObject,
    };
}
//# sourceMappingURL=useObjectFromUrl.js.map