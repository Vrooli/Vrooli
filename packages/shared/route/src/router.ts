import locationHook, { BaseLocationHook, HookNavigationOptions, LocationHook, Path } from "./useLocation";
import makeMatcher, { DefaultParams, Match, MatcherFn } from "./matcher";
import { AnchorHTMLAttributes, cloneElement, createContext, createElement, Fragment, FunctionComponent, isValidElement, useRef, useLayoutEffect, useContext, useCallback, ReactElement, ReactNode, useEffect } from "react";

export type ExtractRouteOptionalParam<PathType extends Path> =
    PathType extends `${infer Param}?`
    ? { [k in Param]: string | undefined }
    : PathType extends `${infer Param}*`
    ? { [k in Param]: string | undefined }
    : PathType extends `${infer Param}+`
    ? { [k in Param]: string }
    : { [k in PathType]: string };

export type ExtractRouteParams<PathType extends string> =
    string extends PathType
    ? { [k in string]: string }
    : PathType extends `${infer _Start}:${infer ParamWithOptionalRegExp}/${infer Rest}`
    ? ParamWithOptionalRegExp extends `${infer Param}(${infer _RegExp})`
    ? ExtractRouteOptionalParam<Param> & ExtractRouteParams<Rest>
    : ExtractRouteOptionalParam<ParamWithOptionalRegExp> &
    ExtractRouteParams<Rest>
    : PathType extends `${infer _Start}:${infer ParamWithOptionalRegExp}`
    ? ParamWithOptionalRegExp extends `${infer Param}(${infer _RegExp})`
    ? ExtractRouteOptionalParam<Param>
    : ExtractRouteOptionalParam<ParamWithOptionalRegExp>
    : {};

export interface RouterProps {
    hook: BaseLocationHook;
    base: Path;
    matcher: MatcherFn;
}

export type NavigationalProps<H extends BaseLocationHook = LocationHook> = (
    | { to: Path; href?: never }
    | { href: Path; to?: never }
) &
    HookNavigationOptions<H>;

export type LinkProps<H extends BaseLocationHook = LocationHook> = Omit<
    AnchorHTMLAttributes<HTMLAnchorElement>,
    "href"
> &
    NavigationalProps<H>;

/*
 * Part 1, Hooks API: useRouter, useRoute and useLocation
 */

// one of the coolest features of `createContext`:
// when no value is provided — default object is used.
// allows us to use the router context as a global ref to store
// the implicitly created router (see `useRouter` below)
const RouterCtx = createContext<{ [x: string]: any }>({});

const buildRouter = ({
    hook = locationHook,
    base = "",
    matcher = makeMatcher(),
} = {}) => ({ hook, base, matcher });

export const useRouter = () => {
    const globalRef = useContext(RouterCtx);

    // either obtain the router from the outer context (provided by the
    // `<Router /> component) or create an implicit one on demand.
    return globalRef.v || (globalRef.v = buildRouter());
};

export const useLocation = () => {
    const router = useRouter();
    return router.hook(router);
};

export const useRoute = <
    T extends DefaultParams | undefined = undefined,
    RoutePath extends Path = Path>(
        pattern: RoutePath
    ):
    Match<T extends DefaultParams ? T : ExtractRouteParams<RoutePath>> => {
    const [path] = useLocation();
    return useRouter().matcher(pattern, path);
};

// internal hook used by Link and Redirect in order to perform navigation
const useNavigate = (options: any) => {
    const navRef = useRef<any>();
    const [, navigate] = useLocation();

    navRef.current = () => navigate(options.to || options.href, options);
    return navRef;
};

/*
 * Part 2, Low Carb Router API: Router, Route, Link, Switch
 */

export const Router = (props: RouterProps): FunctionComponent<Partial<RouterProps> & { children: ReactNode }> => {
    const ref = useRef<any>();

    // this little trick allows to avoid having unnecessary
    // calls to potentially expensive `buildRouter` method.
    // https://reactjs.org/docs/hooks-faq.html#how-to-create-expensive-objects-lazily
    const value = ref.current || (ref.current = { v: buildRouter(props) });

    return createElement(RouterCtx.Provider, {
        value,
        children: (props as any).children,
    }) as any;
};

export type RouteProps = {
    /**
     * If sitemapIndex is true, this specifies the change frequency of the page 
     */
    changeFreq?: "always" | "hourly" | "daily" | "weekly" | "monthly" | "yearly" | "never";
    children: JSX.Element | ((params: any) => JSX.Element);
    component?: any;
    match?: any;
    path?: string;
    /**
     * If sitemapIndex is true, this specifies the priority of the page.
     */
    priority?: number;
    /**
     * Specifies if this route should be included in the sitemap
     */
    sitemapIndex?: boolean;
}

export const Route = ({ path, match, component, children }: RouteProps) => {
    const useRouteMatch = useRoute(path as any);

    // Store last and current path in session storage, so we can handle 
    // dialogs better
    useEffect(() => {
        // Get last stored path in sessionStorage
        const lastPath = sessionStorage.getItem("currentPath");
        // Store last path in sessionStorage
        if (lastPath) sessionStorage.setItem("lastPath", lastPath);
        // Store current path in sessionStorage
        sessionStorage.setItem("currentPath", location.pathname);
    } , [path]);

    // `props.match` is present - Route is controlled by the Switch
    const [matches, params] = match || useRouteMatch;

    if (!matches) return null;

    // React-Router style `component` prop
    if (component) return createElement(component, { params });

    // support render prop or plain children
    return typeof children === "function" ? children(params) : children;
};

export const Link = (props: LinkProps) => {
    const navRef = useNavigate(props);
    const { base } = useRouter();

    let { to, href = to, children, onClick } = props;

    const handleClick = useCallback(
        (event: any) => {
            // ignores the navigation when clicked using right mouse button or
            // by holding a special modifier key: ctrl, command, win, alt, shift
            if (
                event.ctrlKey ||
                event.metaKey ||
                event.altKey ||
                event.shiftKey ||
                event.button !== 0
            )
                return;

            onClick && onClick(event);
            if (!event.defaultPrevented) {
                event.preventDefault();
                navRef.current();
            }
        },
        // navRef is a ref so it never changes
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [onClick]
    );

    // wraps children in `a` if needed
    const extraProps = {
        // handle nested routers and absolute paths
        href: href && href[0] === "~" ? href.slice(1) : base + href,
        onClick: handleClick,
        to: null,
    };
    const jsx = isValidElement(children) ? children : createElement("a", props as any);

    return cloneElement(jsx, extraProps);
};

/**
 * Recursively flattens an array
 * @param children
 * @returns 
 */
const flattenChildren = (children: Array<any> | any): any => {
    return Array.isArray(children)
        ? [].concat(
            ...children.map((c) =>
                c && c.type === Fragment
                    ? flattenChildren(c.props.children)
                    : flattenChildren(c)
            )
        )
        : [children];
};

type SwitchProps = {
    children: JSX.Element | JSX.Element[];
    location?: string;
}

export const Switch = ({ children, location }: SwitchProps) => {
    const { matcher } = useRouter();
    const [originalLocation] = useLocation();

    for (const element of flattenChildren(children)) {
        let match = 0;

        if (
            isValidElement(element) &&
            // we don't require an element to be of type Route,
            // but we do require it to contain a truthy `path` prop.
            // this allows to use different components that wrap Route
            // inside of a switch, for example <AnimatedRoute />.
            (match = (element as any).props.path
                ? matcher((element as any).props.path, location || originalLocation)
                : [true, {}])[0]
        )
            return cloneElement(element, { match } as any);
    }

    return null;
};

export const Redirect = (props: any): ReactElement<any, any> | null => {
    const navRef = useNavigate(props);

    // empty array means running the effect once, navRef is a ref so it never changes
    useLayoutEffect(() => {
        navRef.current();
    }, []); // eslint-disable-line react-hooks/exhaustive-deps

    return null;
};

export default useRoute;