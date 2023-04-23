import { useCallback, useContext, useEffect, useState } from "react";
import { useCustomLazyQuery } from "../../api";
import { userProfile } from "../../api/generated/endpoints/user_profile";
import { getCurrentUser } from "../authentication/session";
import { SessionContext } from "../SessionContext";
export const useProfileQuery = () => {
    const session = useContext(SessionContext);
    const [getData, { data, loading: isProfileLoading }] = useCustomLazyQuery(userProfile, { errorPolicy: "all" });
    useEffect(() => {
        if (getCurrentUser(session).id)
            getData();
    }, [getData, session]);
    const [profile, setProfile] = useState(undefined);
    useEffect(() => {
        if (data)
            setProfile(data);
    }, [data]);
    const onProfileUpdate = useCallback((updatedProfile) => {
        if (updatedProfile)
            setProfile(updatedProfile);
    }, []);
    return {
        onProfileUpdate,
        isProfileLoading,
        profile,
    };
};
//# sourceMappingURL=useProfileQuery.js.map