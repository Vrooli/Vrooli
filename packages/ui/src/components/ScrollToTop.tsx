import { useEffect } from "react";
import { useLocation } from "route";

export const ScrollToTop = () => {
    const [location] = useLocation();
    useEffect(() => {
        if (window.location.hash !== "") {
            setTimeout(() => {
                const id = window.location.hash.replace("#", "");
                const element = document.getElementById(id);
                if (element) {
                    element.scrollIntoView();
                }
            }, 0);
        }
        else window.scrollTo(0, 0);
    }, [location]);
    return null;
};
