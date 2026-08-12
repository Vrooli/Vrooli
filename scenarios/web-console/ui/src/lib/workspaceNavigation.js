import { formatRelativeTime, stripMarkdown } from "../components/MessageJumpList.helpers";
/**
 * Reorder panes so each group occupies ONE contiguous block, anchored at the
 * position of its first member. Ungrouped panes keep their relative positions;
 * a group's members keep their relative order inside the block.
 *
 * This is the invariant the whole navigation layer assumes but nothing used to
 * enforce. Group boundaries are decided purely by adjacency — a group label is
 * emitted whenever `groupId` differs from the previous pane's — so a group
 * whose members are not neighbors renders as several blocks, each with its own
 * header and each claiming the group's full member count. Every ordinary
 * action used to be able to cause that: removing a middle member from a group,
 * dragging any pane through a group's run, or simply reloading (the backend
 * orders by `sort_order, created_at`, and ties are common).
 *
 * Applying this at every write AND at the render boundary makes the split
 * unrepresentable rather than merely unlikely. Returns the input array itself
 * when it already satisfies the invariant, so React sees a stable identity on
 * the overwhelmingly common path.
 */
export function orderPanesByGroupBlocks(panes) {
    const membersByGroup = new Map();
    for (const pane of panes) {
        if (!pane.groupId)
            continue;
        const members = membersByGroup.get(pane.groupId);
        if (members)
            members.push(pane);
        else
            membersByGroup.set(pane.groupId, [pane]);
    }
    if (membersByGroup.size === 0)
        return panes;
    const emitted = new Set();
    const ordered = [];
    for (const pane of panes) {
        const groupId = pane.groupId;
        if (!groupId) {
            ordered.push(pane);
            continue;
        }
        if (emitted.has(groupId))
            continue;
        emitted.add(groupId);
        ordered.push(...(membersByGroup.get(groupId) ?? [pane]));
    }
    return ordered.every((pane, i) => pane === panes[i]) ? panes : ordered;
}
/**
 * The group a pane belongs to after being dropped at `index`, following the
 * tab-group convention users already know from browsers:
 *   - dropped strictly *inside* a group's run → joins that group;
 *   - dropped against an edge of its own group → stays (plain reorder);
 *   - dropped anywhere else → leaves its group.
 *
 * Without this a drop inside a group would be silently undone by
 * `orderPanesByGroupBlocks`, which reads as the drag doing nothing.
 * `panes` must already have the moved pane at `index`.
 */
export function groupIdForDropPosition(panes, index, currentGroupId) {
    const before = panes[index - 1]?.groupId ?? null;
    const after = panes[index + 1]?.groupId ?? null;
    if (before !== null && before === after)
        return before;
    if (currentGroupId !== null && (before === currentGroupId || after === currentGroupId)) {
        return currentGroupId;
    }
    return null;
}
/**
 * View-only reorder of panes for the sidebar. Groups stay contiguous: panes
 * are bucketed by `groupId` in first-appearance order, each bucket is sorted by
 * the chosen comparator, then buckets are concatenated back in block order.
 * "manual" returns the input untouched (stable passthrough).
 */
export function sortPanesForView(panes, sortMode, metrics) {
    if (sortMode === "manual")
        return panes;
    const bucketOrder = [];
    const buckets = new Map();
    for (const pane of panes) {
        const key = pane.groupId;
        let bucket = buckets.get(key);
        if (!bucket) {
            bucket = [];
            buckets.set(key, bucket);
            bucketOrder.push(key);
        }
        bucket.push(pane);
    }
    const empty = { name: "", activityMs: 0, unread: 0, flagged: false };
    const compare = (a, b) => {
        const ma = metrics.get(a.sessionId) ?? empty;
        const mb = metrics.get(b.sessionId) ?? empty;
        switch (sortMode) {
            case "name":
                return ma.name.localeCompare(mb.name);
            case "activity":
                return mb.activityMs - ma.activityMs;
            case "unread":
                // Real unread messages outrank a manual flag (they carry a count and
                // a reason), but a flagged session still sorts above an untouched one.
                return mb.unread - ma.unread
                    || Number(mb.flagged) - Number(ma.flagged)
                    || mb.activityMs - ma.activityMs;
            default:
                return 0;
        }
    };
    const result = [];
    for (const key of bucketOrder) {
        const bucket = buckets.get(key);
        if (!bucket)
            continue;
        result.push(...[...bucket].sort(compare));
    }
    return result;
}
function countUnreadMessages(pane, session) {
    if (!pane.supportsMessagesView || !session)
        return 0;
    return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
}
export function countWorkspaceUnreadMessages(panes, conversationSessions) {
    return panes.reduce((sum, pane) => sum + countUnreadMessages(pane, conversationSessions[pane.sessionId]), 0);
}
function latestEvent(events) {
    return events.reduce((latest, event) => {
        if (!latest)
            return event;
        if (event.sequence !== latest.sequence) {
            return event.sequence > latest.sequence ? event : latest;
        }
        return event.createdAt > latest.createdAt ? event : latest;
    }, null);
}
function eventPreview(event) {
    if (!event?.text)
        return "";
    return stripMarkdown(event.text).replace(/\s+/g, " ").trim();
}
export function buildWorkspaceNavigationItems({ panes, groups, activePane, conversationSessions = {}, viewModes = {}, lastVisitedBySession = {}, now = new Date(), sortMode = "manual", globalIndexBySession, }) {
    const groupMap = new Map(groups.map((group) => [group.id, group]));
    const items = [];
    let lastGroupId = undefined;
    // Group blocks first, in every mode. `sortPanesForView` already partitions
    // by group, so this is a no-op for the non-manual sorts; it is the render
    // boundary's last line of defence for "manual", where the pane array is
    // whatever the store and the backend last agreed on.
    let orderedPanes = orderPanesByGroupBlocks(panes);
    // Non-manual sidebar sorts reorder a view-only copy (sort_order is never
    // touched). Metrics are computed once up front so the comparator is cheap.
    if (sortMode !== "manual") {
        const metrics = new Map();
        for (const pane of panes) {
            const session = conversationSessions[pane.sessionId];
            const latest = latestEvent(session?.events ?? []);
            const activityAt = latest?.createdAt ?? lastVisitedBySession[pane.sessionId] ?? null;
            metrics.set(pane.sessionId, {
                name: pane.name,
                activityMs: activityAt ? Date.parse(activityAt) || 0 : 0,
                unread: countUnreadMessages(pane, session),
                flagged: pane.manuallyUnread,
            });
        }
        orderedPanes = sortPanesForView(orderedPanes, sortMode, metrics);
    }
    orderedPanes.forEach((pane, idx) => {
        const groupId = pane.groupId;
        const group = groupId ? groupMap.get(groupId) : undefined;
        const previousPane = idx > 0 ? orderedPanes[idx - 1] : undefined;
        const nextPane = idx < orderedPanes.length - 1 ? orderedPanes[idx + 1] : undefined;
        const previousInSameGroup = !!group && previousPane?.groupId === groupId;
        const nextInSameGroup = !!group && nextPane?.groupId === groupId;
        if (groupId && groupId !== lastGroupId && group) {
            const tabCount = panes.filter((candidate) => candidate.groupId === groupId).length;
            items.push({ kind: "group-label", group, tabCount });
        }
        lastGroupId = groupId;
        if (group?.isCollapsed)
            return;
        const session = conversationSessions[pane.sessionId];
        const latest = latestEvent(session?.events ?? []);
        const latestEventAt = latest?.createdAt ?? null;
        const lastVisitedAt = lastVisitedBySession[pane.sessionId] ?? null;
        const activityAt = latestEventAt ?? lastVisitedAt;
        const activityLabel = activityAt
            ? latestEventAt
                ? formatRelativeTime(activityAt, now)
                : `Visited ${formatRelativeTime(activityAt, now)}`
            : "";
        const unreadCount = countUnreadMessages(pane, session);
        items.push({
            kind: "pane",
            pane,
            globalIndex: globalIndexBySession?.[pane.sessionId] ?? idx,
            group,
            groupPosition: group
                ? previousInSameGroup && nextInSameGroup
                    ? "middle"
                    : previousInSameGroup
                        ? "last"
                        : nextInSameGroup
                            ? "first"
                            : "single"
                : undefined,
            isActive: pane.sessionId === activePane,
            unreadCount,
            viewMode: pane.supportsMessagesView ? (viewModes[pane.sessionId] ?? "terminal") : "terminal",
            latestEventAt,
            lastVisitedAt,
            activityLabel,
            previewText: eventPreview(latest),
        });
    });
    return items;
}
// ---------------------------------------------------------------------------
// Origin buckets
// ---------------------------------------------------------------------------
/** Display order of the sidebar origin tabs, top to bottom / left to right. */
export const ORIGIN_BUCKET_ORDER = ["ui", "programmatic", "remote"];
/**
 * Fold a session's provenance into a sidebar origin bucket. "unspecified" (and
 * any unknown/absent origin) normalizes to "programmatic", matching the server's
 * normalization of an origin-less create — so a session that reaches the UI
 * untagged still lands in a real bucket rather than a fourth phantom one.
 */
export function originBucket(origin) {
    return origin === "ui" ? "ui" : origin === "remote" ? "remote" : "programmatic";
}
/**
 * Partition the sidebar into origin buckets (UI-owned / Programmatic / Remote),
 * each carrying its own navigation list with groups, sort, and drag intact.
 *
 * Only non-empty buckets are returned, in ORIGIN_BUCKET_ORDER. Each bucket's
 * items are built over just that bucket's panes — so groups, sort modes, and
 * group-position rounding compose *within* the bucket — while `globalIndex` is
 * pinned to the pane's position in the full (unbucketed) list so drag-reorder
 * still addresses the backing store array.
 *
 * A group is atomic here: every member follows the group's FIRST member into a
 * single bucket, whatever its own provenance. A group is a unit the user made
 * by hand, so a mixed-origin group (say, one session opened in the UI and one
 * started by an agent) must not be torn in half across two tabs — where each
 * half looks like the group has silently lost members.
 *
 * With only UI-origin sessions this returns a single "ui" bucket, and the caller
 * renders that list exactly as it did before origin tabs existed.
 */
export function buildOriginBucketedNavigation({ originBySession, ...options }) {
    const { panes } = options;
    const globalIndexBySession = {};
    panes.forEach((pane, index) => {
        globalIndexBySession[pane.sessionId] = index;
    });
    const bucketByGroup = new Map();
    for (const pane of panes) {
        if (!pane.groupId || bucketByGroup.has(pane.groupId))
            continue;
        bucketByGroup.set(pane.groupId, originBucket(originBySession[pane.sessionId]));
    }
    const bucketForPane = (pane) => (pane.groupId ? bucketByGroup.get(pane.groupId) : undefined)
        ?? originBucket(originBySession[pane.sessionId]);
    const result = [];
    for (const bucket of ORIGIN_BUCKET_ORDER) {
        const bucketPanes = panes.filter((pane) => bucketForPane(pane) === bucket);
        if (bucketPanes.length === 0)
            continue;
        result.push({
            bucket,
            items: buildWorkspaceNavigationItems({ ...options, panes: bucketPanes, globalIndexBySession }),
        });
    }
    return result;
}
