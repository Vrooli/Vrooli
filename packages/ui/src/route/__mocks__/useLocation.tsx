const defaultLocation = {
    pathname: "/",
    search: "",
    hash: "",
    state: undefined,
};

let location = { ...defaultLocation };

function setLocation(newLocation) {
    location = { ...location, ...newLocation };
    // Update window location properties as needed
    window.history.pushState({}, "", location.pathname + location.search + location.hash);
}

export function useLocation() {
    return [location, setLocation];
}

