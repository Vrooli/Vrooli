import { SessionUser } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { userSchedulePartial } from "./userSchedule";

export const sessionUserPartial: GqlPartial<SessionUser> = {
    __typename: 'SessionUser',
    full: {
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        schedules: () => relPartial(userSchedulePartial, 'full'),
        theme: true,
    }
}