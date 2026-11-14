/*
 * Notification Hub explainer bridge. Runs INSIDE the sealed frame (sandbox
 * "allow-scripts", opaque origin). The page talks to the Hub only through
 * postMessage; the Hub validates every message before it acts on it
 * (bridge.ts). Plain ES2019, no dependencies.
 *
 *   hub.open(name, path?)            open an attachment in the Hub's viewer
 *   hub.prefill(questionId, value)   fill one question of the Hub form (the
 *                                    operator still sends the answer)
 *   hub.theme                        "light" | "dark"
 *   [data-hub-open="name"]           tap = hub.open("name"); "name/path/in/folder" also works
 */
(function () {
  "use strict";
  if (window.hub) return;

  var PROTOCOL = 1;
  var NAME = /^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$/;
  var root = document.documentElement;

  function post(message) {
    message.hub = PROTOCOL;
    window.parent.postMessage(message, "*");
  }

  function open(name, path) {
    if (typeof name !== "string" || !NAME.test(name)) return;
    var message = { type: "open", name: name };
    if (typeof path === "string" && path) message.path = path;
    post(message);
  }

  function prefill(questionId, value) {
    if (typeof questionId !== "string" || !questionId) return;
    post({ type: "prefill", questionId: questionId, value: value });
  }

  var hub = { open: open, prefill: prefill };
  Object.defineProperty(hub, "theme", {
    enumerable: true,
    get: function () {
      return root.getAttribute("data-theme") === "dark" ? "dark" : "light";
    },
  });
  Object.defineProperty(window, "hub", { value: Object.freeze(hub), configurable: false, writable: false });

  function openTarget(element) {
    var value = element.getAttribute("data-hub-open") || "";
    var slash = value.indexOf("/");
    if (slash < 0) open(value);
    else open(value.slice(0, slash), value.slice(slash + 1));
  }

  document.addEventListener(
    "click",
    function (event) {
      var target = event.target;
      var element = target && target.closest ? target.closest("[data-hub-open]") : null;
      if (!element) return;
      event.preventDefault();
      openTarget(element);
    },
    true,
  );

  document.addEventListener("keydown", function (event) {
    if (event.key !== "Enter" && event.key !== " ") return;
    var target = event.target;
    var element = target && target.closest ? target.closest("[data-hub-open]") : null;
    // Links and buttons already turn Enter and Space into a click.
    if (!element || element.matches("a[href], button, input, select, textarea")) return;
    event.preventDefault();
    openTarget(element);
  });

  // The Hub tells the page when the operator switches between light and dark.
  window.addEventListener("message", function (event) {
    if (event.source !== window.parent) return;
    var data = event.data;
    if (!data || data.hub !== PROTOCOL || data.type !== "theme") return;
    root.setAttribute("data-theme", data.theme === "dark" ? "dark" : "light");
    window.dispatchEvent(new CustomEvent("hubthemechange", { detail: hub.theme }));
  });

  // The frame has no scrollbar of its own: it reports its content height and
  // the Hub sizes the frame to it.
  var lastHeight = 0;
  var frame = 0;
  function reportHeight() {
    frame = 0;
    var height = Math.ceil(root.getBoundingClientRect().height);
    if (height > 0 && height !== lastHeight) {
      lastHeight = height;
      post({ type: "height", height: height });
    }
  }
  function scheduleHeight() {
    if (!frame) frame = window.requestAnimationFrame(reportHeight);
  }
  if (typeof ResizeObserver === "function") new ResizeObserver(scheduleHeight).observe(root);
  window.addEventListener("load", scheduleHeight);
  document.addEventListener("DOMContentLoaded", scheduleHeight);
  document.addEventListener("load", scheduleHeight, true);
  scheduleHeight();
})();
