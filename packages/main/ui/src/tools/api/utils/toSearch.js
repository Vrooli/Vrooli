import { rel } from "./rel";
export const toSearch = async (partial) => {
    const { __define, ...node } = await rel(partial, "list");
    return [{
            __typename: `${partial.__typename}SearchResult`,
            list: {
                __define,
                edges: {
                    cursor: true,
                    node,
                },
                pageInfo: {
                    endCursor: true,
                    hasNextPage: true,
                },
            },
        }, "list"];
};
//# sourceMappingURL=toSearch.js.map