export type GetSearchStringProps = {
    objectType: GraphQLModelType;
    searchString: string;
    languages?: string[] | undefined;
}