package com.vrooli.portal.tv;
import java.util.*;
/** Dependency-free JVM checks for the actual native navigation model. */
public final class NavigationStateTest {
    private static void equal(Object expected, Object actual, String message) {
        if (!Objects.equals(expected, actual)) throw new AssertionError(message + ": expected " + expected + ", got " + actual);
    }
    public static void main(String[] args) {
        NavigationState state = new NavigationState();
        state.navigate("Vrooli"); equal("Home", state.back(), "Library Back returns to Home");
        state.navigate("Settings"); state.navigate("Vrooli"); equal("Home", state.back(), "Reconnect cannot create Settings/library Back cycle");
        state.page = "TV apps"; state.navigate("Vrooli"); equal("TV apps", state.back(), "Library remembers its real entry page");
        state.navigate("Settings"); equal("Vrooli", state.back(), "Cancel settings returns to library");
        state.navigate("Vrooli"); equal("TV apps", state.back(), "Reconnect preserves existing library return");
        state.focused.put("Vrooli", "experience:nutrition-planner"); state.filter = "Nooch"; state.scroll.put("Vrooli", 220);
        state.navigate("Home"); state.navigate("Vrooli");
        equal("experience:nutrition-planner", state.focused.get("Vrooli"), "Tab entry preserves card identity");
        equal("Nooch", state.filter, "Tab entry preserves search"); equal(220, state.scroll.get("Vrooli"), "Tab entry preserves scroll");
        equal(Arrays.asList("a", "b"), NavigationState.favorites(Arrays.asList("a", "b", "c"), Arrays.asList("missing", "a", "a", "b", "c"), 2), "Installed unique favorites keep explicit order and bound");
        equal(Collections.emptyList(), NavigationState.favorites(Arrays.asList("a"), Collections.emptyList(), 4), "Intentionally empty favorites stay empty");
        List<String> order = new ArrayList<>(Arrays.asList("a", "b", "c"));
        equal(false, NavigationState.move(order, "a", -1), "First favorite cannot move left");
        equal(true, NavigationState.move(order, "c", -1), "Favorite moves left");
        equal(Arrays.asList("a", "c", "b"), order, "Move preserves neighboring order");
        equal(false, NavigationState.move(order, "missing", 1), "Unknown favorite cannot be inserted by reorder");
        equal(Arrays.asList("c", "b", "a", "d"), NavigationState.homeApps(Arrays.asList("a", "b", "c", "d"), Arrays.asList("gone", "c", "c", "b"), Arrays.asList("b", "a")), "Recent clicks lead, pins follow, all remaining apps stay reachable");
        equal(Arrays.asList("b", "a", "c"), NavigationState.homeApps(Arrays.asList("a", "b", "c"), Collections.emptyList(), Arrays.asList("b", "gone")), "Cleared history keeps pins then installed order");
        List<String> many = new ArrayList<>(); for (int i = 0; i < 50; i++) many.add("app" + i);
        List<String> shortcuts = NavigationState.homeApps(many, Arrays.asList("app49"), Arrays.asList("app48"));
        equal(12, shortcuts.size(), "Home keeps end destinations reachable with a large installed inventory");
        equal(Arrays.asList("app49", "app48"), shortcuts.subList(0, 2), "Bounded shortcuts still put recent use and pins first");
        for (String previous : Arrays.asList("Home", "Vrooli", "TV apps", "Inputs", "Settings")) {
            state.page = previous; state.libraryReturn = "Settings"; state.settingsReturn = "Vrooli";
            state.systemHome();
            equal("Home", state.page, "System Home returns from " + previous);
            equal("Home", state.back(), "System Home resets return navigation");
            equal("Nooch", state.filter, "System Home preserves useful library query");
            equal(false, state.exitsOnBack(true), "Daily Home cannot exit on Back");
            equal(true, state.exitsOnBack(false), "Ordinary preview can exit from Home");
        }
        System.out.println("PASS: navigation returns, reconnect, focus/filter/scroll, favorites and reorder");
    }
}
