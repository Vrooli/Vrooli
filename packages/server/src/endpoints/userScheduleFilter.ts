import { gql } from 'apollo-server-express';
import { UserScheduleFilterType } from '@shared/consts';

export const typeDef = gql`
    enum UserScheduleFilterType {
        Blur
        Hide
        ShowMore
    }

    input UserScheduleFilterCreateInput {
        id: ID!
        filterType: UserScheduleFilterType!
        userScheduleConnect: ID!
        tagConnect: ID
        tagCreate: TagCreateInput
    }
    type UserScheduleFilter {
        id: ID!
        filterType: UserScheduleFilterType!
        userSchedule: UserSchedule!
        tag: Tag
    }
`

export const resolvers: {
    UserScheduleFilterType: typeof UserScheduleFilterType;
} = {
    UserScheduleFilterType,
}