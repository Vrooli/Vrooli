import { FocusModeFilterType } from "@local/shared";

export const typeDef = `#graphql
    enum FocusModeFilterType {
        Blur
        Hide
        ShowMore
    }

    input FocusModeFilterCreateInput {
        id: ID!
        filterType: FocusModeFilterType!
        focusModeConnect: ID!
        tagConnect: ID
        tagCreate: TagCreateInput
    }
    type FocusModeFilter {
        id: ID!
        filterType: FocusModeFilterType!
        focusMode: FocusMode!
        tag: Tag!
    }
`;

export const resolvers: {
    FocusModeFilterType: typeof FocusModeFilterType;
} = {
    FocusModeFilterType,
};
