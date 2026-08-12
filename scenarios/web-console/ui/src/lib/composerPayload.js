/**
 * Compose the single outbound payload from the typed text and the resolved
 * attachment paths. Injection format (locked with the operator): text first,
 * then the space-joined resolved paths in attachment order, joined by a single
 * space, with NO forced trailing newline. web-console uploads target a raw
 * terminal, so this cannot copy swarm-manager's structured format 1:1.
 */
export function composeComposerPayload(text, paths) {
    if (paths.length === 0)
        return text;
    const joined = paths.join(" ");
    return text ? `${text} ${joined}` : joined;
}
