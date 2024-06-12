if (typeof setImmediate === 'undefined') {
    global.setImmediate = (fn) => setTimeout(fn, 0);
}