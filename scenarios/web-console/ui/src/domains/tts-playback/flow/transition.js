import { nextPlaybackTransportStatus } from "./generated/runtime";
export { playbackTransportEvents, playbackTransportStatuses, isPlaybackTransportEventValid, nextPlaybackTransportStatus, transitionPlaybackTransportStatus, } from "./generated/runtime";
export const initialPlaybackTransportState = { status: "idle" };
const carryPayload = (state) => {
    switch (state.status) {
        case "loading":
        case "playing":
        case "paused":
            return state;
        case "error":
            return null;
        case "idle":
            return null;
    }
};
export const checkPlaybackTransportInvariants = (state) => {
    if (state.status === "idle")
        return;
    if (state.status === "error") {
        if (state.message.trim() === "") {
            throw new Error("error playback state requires a message");
        }
    }
    else if (state.loadId.trim() === "") {
        throw new Error(`${state.status} playback state requires a loadId`);
    }
    if (state.status === "loading" || state.status === "playing" || state.status === "paused" || state.status === "error") {
        if (state.eventId.trim() === "") {
            throw new Error(`${state.status} playback state requires an eventId`);
        }
        if (state.sessionId.trim() === "") {
            throw new Error(`${state.status} playback state requires a sessionId`);
        }
        if (state.queueIndex < 0 || state.queueIndex >= state.queue.length) {
            throw new Error(`${state.status} queueIndex ${state.queueIndex} out of range [0, ${state.queue.length})`);
        }
        if (state.queue[state.queueIndex] !== state.eventId) {
            throw new Error(`${state.status} eventId ${state.eventId} does not match queue[${state.queueIndex}]`);
        }
    }
};
const assertNoDrift = (fromStatus, eventType, next, matchesFormalStatus) => {
    if (!matchesFormalStatus)
        return;
    const expectedStatus = nextPlaybackTransportStatus(fromStatus, eventType);
    if (expectedStatus !== next.status) {
        throw new Error(`playback transport transition drift: ${fromStatus}/${eventType} produced ${next.status}, want ${expectedStatus}`);
    }
};
export const transitionPlaybackTransport = (state, event) => {
    let next;
    let matchesFormalStatus = true;
    switch (event.type) {
        case "play": {
            if (event.queue.length === 0) {
                throw new Error("playback transport play event requires a non-empty queue");
            }
            if (event.queueIndex < 0 || event.queueIndex >= event.queue.length) {
                throw new Error(`playback transport play queueIndex out of range: ${event.queueIndex}`);
            }
            if (event.queue[event.queueIndex] !== event.eventId) {
                throw new Error("playback transport play eventId must equal queue[queueIndex]");
            }
            next = {
                status: "loading",
                sessionId: event.sessionId,
                eventId: event.eventId,
                loadId: event.loadId,
                queue: event.queue,
                queueIndex: event.queueIndex,
            };
            break;
        }
        case "loadResolved": {
            if (state.status === "loading") {
                if (state.loadId === event.loadId) {
                    next = { ...state, status: "playing" };
                }
                else {
                    next = state;
                    matchesFormalStatus = false;
                }
            }
            else {
                next = state;
            }
            break;
        }
        case "loadFailed": {
            if (state.status === "loading") {
                if (state.loadId === event.loadId) {
                    next = {
                        status: "error",
                        sessionId: state.sessionId,
                        eventId: state.eventId,
                        message: event.message,
                        queue: state.queue,
                        queueIndex: state.queueIndex,
                    };
                }
                else {
                    next = state;
                    matchesFormalStatus = false;
                }
            }
            else {
                next = state;
            }
            break;
        }
        case "trackStarted": {
            if (state.status === "playing" && state.eventId === event.eventId) {
                next = state;
            }
            else {
                next = state;
            }
            break;
        }
        case "trackEnded": {
            if (state.status === "playing") {
                if (state.eventId === event.eventId) {
                    next = { status: "idle" };
                }
                else {
                    next = state;
                    matchesFormalStatus = false;
                }
            }
            else {
                next = state;
            }
            break;
        }
        case "pause": {
            if (state.status === "playing") {
                next = { ...state, status: "paused" };
            }
            else {
                next = state;
            }
            break;
        }
        case "resume": {
            if (state.status === "paused") {
                next = { ...state, status: "playing" };
            }
            else {
                next = state;
            }
            break;
        }
        case "stop": {
            next = { status: "idle" };
            break;
        }
        case "playbackError": {
            const carry = carryPayload(state);
            if (carry === null) {
                next = state;
            }
            else {
                next = {
                    status: "error",
                    sessionId: carry.sessionId,
                    eventId: carry.eventId,
                    message: event.message,
                    queue: carry.queue,
                    queueIndex: carry.queueIndex,
                };
            }
            break;
        }
    }
    assertNoDrift(state.status, event.type, next, matchesFormalStatus);
    checkPlaybackTransportInvariants(next);
    return next;
};
export const isPlaybackPlaying = (state) => state.status === "playing";
export const isPlaybackPaused = (state) => state.status === "paused";
export const isPlaybackBusy = (state) => state.status === "loading" || state.status === "playing" || state.status === "paused";
export const hasNextInQueue = (state) => {
    if (state.status === "idle" || state.status === "error") {
        return state.status === "error" ? state.queueIndex + 1 < state.queue.length : false;
    }
    return state.queueIndex + 1 < state.queue.length;
};
export const buildPlayNextEvent = (state, loadId) => {
    if (state.status === "idle")
        return null;
    const nextIndex = state.queueIndex + 1;
    if (nextIndex >= state.queue.length)
        return null;
    const nextEventId = state.queue[nextIndex];
    if (!nextEventId)
        return null;
    return {
        type: "play",
        sessionId: state.sessionId,
        eventId: nextEventId,
        loadId,
        queue: state.queue,
        queueIndex: nextIndex,
    };
};
