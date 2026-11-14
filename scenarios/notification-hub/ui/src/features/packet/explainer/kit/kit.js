/*
 * Notification Hub explainer starter kit: behaviour.
 *
 * Plain ES2019, no dependencies, safe to run more than once. It owns only the
 * page behaviours below; the Hub's bridge (hub.open, hub.prefill, hub.theme) is
 * injected separately.
 *
 *   [data-hub-compare]  before/after slider over its first two children.
 *       data-before-label / data-after-label   tag text (default "Before" / "After")
 *       data-position                          start position, 0 to 100 (default 50)
 *
 * The slider's range input is created here, so authored HTML needs no form control.
 * Call `window.hubKit.init(root)` after adding compare blocks to the page later.
 */
(function () {
  "use strict";

  var READY = "data-hub-ready";

  function clamp(value) {
    return Math.min(100, Math.max(0, value));
  }

  function tag(text, side) {
    var el = document.createElement("span");
    el.className = "hub-compare__tag is-" + side;
    el.textContent = text;
    el.setAttribute("aria-hidden", "true");
    return el;
  }

  function initCompare(root) {
    if (root.hasAttribute(READY)) return;
    var layers = Array.prototype.slice.call(root.children, 0, 2);
    if (layers.length < 2) return;

    var beforeLabel = root.getAttribute("data-before-label") || "Before";
    var afterLabel = root.getAttribute("data-after-label") || "After";
    var start = parseFloat(root.getAttribute("data-position"));

    root.classList.add("hub-compare");
    layers[0].classList.add("hub-compare__layer", "hub-compare__before");
    layers[1].classList.add("hub-compare__layer", "hub-compare__after");
    layers[0].setAttribute("aria-label", beforeLabel);
    layers[1].setAttribute("aria-label", afterLabel);
    layers[0].setAttribute("role", "group");
    layers[1].setAttribute("role", "group");

    var range = document.createElement("input");
    range.type = "range";
    range.min = "0";
    range.max = "100";
    range.step = "1";
    range.className = "hub-compare__range";
    range.setAttribute("aria-label", beforeLabel + " and " + afterLabel);

    var handle = document.createElement("span");
    handle.className = "hub-compare__handle";
    handle.setAttribute("aria-hidden", "true");

    root.appendChild(tag(beforeLabel, "before"));
    root.appendChild(tag(afterLabel, "after"));
    root.appendChild(range);
    root.appendChild(handle);

    function setPosition(value) {
      var pos = clamp(value);
      range.value = String(pos);
      root.style.setProperty("--hub-pos", pos + "%");
      range.setAttribute("aria-valuetext", Math.round(100 - pos) + "% " + afterLabel);
    }

    function fromPointer(event) {
      var box = root.getBoundingClientRect();
      if (box.width > 0) setPosition(((event.clientX - box.left) / box.width) * 100);
    }

    var dragging = false;
    root.addEventListener("pointerdown", function (event) {
      if (event.button > 0) return;
      dragging = true;
      if (root.setPointerCapture && event.pointerId !== undefined) {
        try { root.setPointerCapture(event.pointerId); } catch (e) { /* synthetic pointer */ }
      }
      fromPointer(event);
    });
    root.addEventListener("pointermove", function (event) {
      if (dragging) fromPointer(event);
    });
    function stop() { dragging = false; }
    root.addEventListener("pointerup", stop);
    root.addEventListener("pointercancel", stop);
    range.addEventListener("input", function () {
      setPosition(parseFloat(range.value));
    });

    setPosition(isNaN(start) ? 50 : start);
    root.setAttribute(READY, "");
  }

  function init(scope) {
    var nodes = (scope || document).querySelectorAll("[data-hub-compare]");
    for (var i = 0; i < nodes.length; i++) initCompare(nodes[i]);
  }

  if (window.hubKit && typeof window.hubKit.init === "function") {
    window.hubKit.init();
    return;
  }
  window.hubKit = { init: init };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { init(); });
  } else {
    init();
  }
})();
