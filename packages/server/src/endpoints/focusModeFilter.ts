import { FocusModeFilterType } from '@shared/consts';
import { gql } from 'apollo-server-express';

export const typeDef = gql`
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
`

export const resolvers: {
    FocusModeFilterType: typeof FocusModeFilterType;
} = {
    FocusModeFilterType,
}