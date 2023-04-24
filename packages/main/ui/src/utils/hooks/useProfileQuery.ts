import { User } from "@local/shared";
import { useCallback, useContext, useEffect, useState } from "react";
import { useCustomLazyQuery } from "../../api";
import { userProfile } from "../../api/generated/endpoints/user_profile";
import { SessionContext } from "../SessionContext";
import { getCurrentUser } from "../authentication/session";

export const useProfileQuery = () => {
    const session = useContext(SessionContext);
    const [getData, { data, loading: isProfileLoading }] = useCustomLazyQuery<User, undefined>(userProfile, { errorPolicy: "all" });
    useEffect(() => {
        if (getCurrentUser(session).id) getData();
    }, [getData, session]);
    const [profile, setProfile] = useState<User | undefined>(undefined);
    useEffect(() => {
        if (data) setProfile(data);
    }, [data]);
    const onProfileUpdate = useCallback((updatedProfile: User | undefined) => {
        if (updatedProfile) setProfile(updatedProfile);
    }, []);

    return {
        onProfileUpdate,
        isProfileLoading,
        profile,
    };
};
