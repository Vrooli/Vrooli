import { cloneElement, createContext, createElement, Fragment, isValidElement, Suspense, useCallback, useContext, useEffect, useLayoutEffect, useRef } from "react";
import makeMatcher from "./matcher";
import { parseSearchParams } from "./searchParams";
import locationHook from "./useLocation";
const RouterCtx = createContext({});
const buildRouter = ({ hook = locationHook, base = "", matcher = makeMatcher(), } = {}) => ({ hook, base, matcher });
export const useRouter = () => {
    const globalRef = useContext(RouterCtx);
    return globalRef.v || (globalRef.v = buildRouter());
};
export const useLocation = () => {
    const router = useRouter();
    return router.hook(router);
};
export const useRoute = (pattern) => {
    const [path] = useLocation();
    return useRouter().matcher(pattern, path);
};
const useNavigate = (options) => {
    const navRef = useRef();
    const [, navigate] = useLocation();
    navRef.current = () => navigate(options.to || options.href, options);
    return navRef;
};
export const Router = (props) => {
    const ref = useRef();
    const value = ref.current || (ref.current = { v: buildRouter(props) });
    return createElement(RouterCtx.Provider, {
        value,
        children: props.children,
    });
};
export const Route = ({ path, match, component, children }) => {
    const useRouteMatch = useRoute(path);
    useEffect(() => {
        const lastCurrentPath = sessionStorage.getItem("currentPath");
        const lastCurrentSearchParams = sessionStorage.getItem("currentSearchParams");
        if (!lastCurrentPath)
            sessionStorage.setItem("currentPath", window.location.pathname);
        if (!lastCurrentSearchParams)
            sessionStorage.setItem("currentSearchParams", JSON.stringify(parseSearchParams()));
    }, [path]);
    const [matches, params] = match || useRouteMatch;
    if (!matches)
        return null;
    if (component)
        return createElement(component, { params });
    return typeof children === "function" ? children(params) : children;
};
export const Link = (props) => {
    const navRef = useNavigate(props);
    const { base } = useRouter();
    const { to, href = to, children, onClick } = props;
    const handleClick = useCallback((event) => {
        if (event.ctrlKey ||
            event.metaKey ||
            event.altKey ||
            event.shiftKey ||
            event.button !== 0)
            return;
        onClick && onClick(event);
        if (!event.defaultPrevented) {
            event.preventDefault();
            navRef.current();
        }
    }, [onClick]);
    const extraProps = {
        href: href && href[0] === "~" ? href.slice(1) : base + href,
        onClick: handleClick,
        to: null,
    };
    const jsx = isValidElement(children) ? children : createElement("a", props);
    return cloneElement(jsx, extraProps);
};
const flattenChildren = (children) => {
    return Array.isArray(children)
        ? [].concat(...children.map((c) => c && c.type === Fragment
            ? flattenChildren(c.props.children)
            : flattenChildren(c)))
        : [children];
};
export const Switch = ({ children, location, fallback }) => {
    const { matcher } = useRouter();
    const [originalLocation] = useLocation();
    for (const element of flattenChildren(children)) {
        let match = 0;
        if (isValidElement(element) &&
            (match = element.props.path
                ? matcher(element.props.path, location || originalLocation)
                : [true, {}])[0]) {
            if (fallback) {
                return createElement(Suspense, { fallback }, cloneElement(element, { match }));
            }
            return cloneElement(element, { match });
        }
    }
    return null;
};
export const Redirect = (props) => {
    const navRef = useNavigate(props);
    useLayoutEffect(() => {
        navRef.current();
    }, []);
    return null;
};
export default useRoute;
//# sourceMappingURL=router.js.map