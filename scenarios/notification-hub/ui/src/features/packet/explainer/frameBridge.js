/*
 * Notification Hub explainer bridge. Runs INSIDE the sealed frame (sandbox
 * "allow-scripts", opaque origin). The page talks to the Hub only through
 * postMessage; the Hub validates every message before it acts on it
 * (bridge.ts). Plain ES2019, no dependencies.
 *
 *   hub.open(name, path?)            open an attachment in the Hub's viewer
 *   hub.prefill(questionId, value)   fill one question of the Hub form (the
 *                                    operator still sends the answer)
 *   hub.answer.set(key,label,value)  put one decision in the Hub answer bar
 *   hub.answer.remove(key)           remove one decision from that bar
 *   hub.note(key, label)              open the Hub-owned note sheet for an item
 *   hub.openAsk(id)                   navigate to another Hub ask
 *   hub.asset("folder/file.png")     resolve a Hub attachment for a viewer
 *   hub.context                       read-only kind, recurrence and named asks
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

  // Touch events in a sealed iframe do not bubble to the host document.
  var swipeStart = null;
  document.addEventListener("touchstart", function (event) {
    var target = event.target;
    if (event.touches.length !== 1 || (target && target.closest && target.closest("input, textarea, select, [contenteditable=true], [data-no-packet-swipe]"))) { swipeStart = null; return; }
    swipeStart = { x: event.touches[0].clientX, y: event.touches[0].clientY };
  }, { passive: true });
  document.addEventListener("touchend", function (event) {
    if (!swipeStart || event.changedTouches.length !== 1) return;
    var dx = event.changedTouches[0].clientX - swipeStart.x;
    var dy = event.changedTouches[0].clientY - swipeStart.y;
    swipeStart = null;
    if (Math.abs(dx) >= 80 && Math.abs(dx) >= Math.abs(dy) * 1.5) post({ type: "swipe", direction: dx > 0 ? "previous" : "next" });
  }, { passive: true });

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

  function note(key, label) {
    if (typeof key !== "string" || !key.trim() || typeof label !== "string" || !label.trim()) return;
    post({ type: "note", key: key, label: label });
  }

  function openAsk(askId) {
    if (typeof askId !== "string" || !askId) return;
    post({ type: "open-ask", askId: askId });
  }

  var assetSequence = 0;
  var assetRequests = Object.create(null);
  function asset(reference) {
    if (typeof reference !== "string" || reference.length > 512) return Promise.resolve(null);
    var slash = reference.indexOf("/");
    if (slash < 1 || slash === reference.length - 1) return Promise.resolve(null);
    var name = reference.slice(0, slash);
    var path = reference.slice(slash + 1);
    if (!NAME.test(name) || path.split("/").some(function (part) { return !part || part === "." || part === ".."; })) return Promise.resolve(null);
    assetSequence = (assetSequence % 99999999) + 1;
    var requestId = "a" + assetSequence;
    return new Promise(function (resolve) {
      var timeout = window.setTimeout(function () {
        delete assetRequests[requestId];
        resolve(null);
      }, 15000);
      assetRequests[requestId] = { resolve: resolve, timeout: timeout };
      post({ type: "asset", requestId: requestId, name: name, path: path });
    });
  }

  var answer = {
    set: function (key, label, value) {
      if (typeof key !== "string" || typeof label !== "string") return;
      post({ type: "answer", operation: "set", key: key, label: label, value: value });
    },
    remove: function (key) {
      if (typeof key !== "string") return;
      post({ type: "answer", operation: "remove", key: key });
    },
  };
  var context = {};
  try {
    var contextNode = document.getElementById("hub-context");
    if (contextNode) context = JSON.parse(contextNode.textContent || "{}");
  } catch (_) {}
  function freeze(value) {
    if (!value || typeof value !== "object" || Object.isFrozen(value)) return value;
    Object.keys(value).forEach(function (key) { freeze(value[key]); });
    return Object.freeze(value);
  }
  freeze(context);

  var hub = { open: open, openAsk: openAsk, note: note, asset: asset, prefill: prefill, answer: Object.freeze(answer), context: context,
    choiceCapability: function (interactive) {
      if (typeof interactive === "boolean") window.parent.postMessage({ type: "hub:answer-capability", interactive: interactive }, "*");
    },
  };
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
  // A page whose own colours work in one scheme only (data-hub-scheme) stays in it.
  window.addEventListener("message", function (event) {
    if (event.source !== window.parent) return;
    var data = event.data;
    if (data && data.hub === PROTOCOL && data.type === "asset-resolve" && typeof data.requestId === "string" && Object.prototype.hasOwnProperty.call(assetRequests, data.requestId)) {
      var request = assetRequests[data.requestId];
      delete assetRequests[data.requestId];
      window.clearTimeout(request.timeout);
      request.resolve(typeof data.dataUrl === "string" ? data.dataUrl : null);
      return;
    }
    if (data && data.hub === PROTOCOL && data.type === "answer-restore" && data.changes && typeof data.changes === "object") {
      window.dispatchEvent(new CustomEvent("hub:answer-restore", { detail: { changes: data.changes } }));
      return;
    }
    if (data && data.hub === PROTOCOL && data.type === "answer-remove" && typeof data.key === "string") {
      window.dispatchEvent(new CustomEvent("hub:answer-remove", { detail: { key: data.key } }));
      return;
    }
    if (!data || data.hub !== PROTOCOL || data.type !== "theme") return;
    if (root.hasAttribute("data-hub-scheme")) return;
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
