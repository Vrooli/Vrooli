const defaultLocation = {
    pathname: "/",
    search: "",
    hash: "",
    state: undefined,
};

let location = { ...defaultLocation };

const setLocation = (newLocation) => {
    location = { ...location, ...newLocation };
    // Update window location properties as needed
    window.history.pushState({}, "", location.pathname + location.search + location.hash);
};

export const useLocation = () => [location, setLocation];

