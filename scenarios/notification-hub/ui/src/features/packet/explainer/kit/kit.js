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
 *   [data-hub-groups="<question id>"]  grouped items; each .hub-group[data-key]
 *       has a .hub-group__toggle button (aria-pressed). A tap flips it and fills
 *       the Hub form's multi_choice question with the keys switched on, through
 *       hub.prefill. The operator still sends.
 *   [data-hub-copy]     a button that copies the text of the nearest <code> in
 *       its parent. A sealed frame may refuse the clipboard; the command text is
 *       then selected so one long-press copies it, and the button says so.
 *
 * The slider's range input is created here, so authored HTML needs no form control.
 * Call `window.hubKit.init(root)` after adding blocks to the page later.
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

  function selectedKeys(root) {
    var keys = [];
    var groups = root.querySelectorAll(".hub-group[data-key]");
    for (var i = 0; i < groups.length; i++) {
      var toggle = groups[i].querySelector(".hub-group__toggle");
      if (toggle && toggle.getAttribute("aria-pressed") === "true") keys.push(groups[i].getAttribute("data-key"));
    }
    return keys;
  }

  function initGroups(root) {
    if (root.hasAttribute(READY)) return;
    var question = root.getAttribute("data-hub-groups");
    root.addEventListener("click", function (event) {
      var toggle = event.target && event.target.closest ? event.target.closest(".hub-group__toggle") : null;
      if (!toggle || !root.contains(toggle)) return;
      toggle.setAttribute("aria-pressed", toggle.getAttribute("aria-pressed") === "true" ? "false" : "true");
      if (question && window.hub && window.hub.prefill) window.hub.prefill(question, { selected: selectedKeys(root) });
    });
    root.setAttribute(READY, "");
  }

  function selectText(node) {
    var selection = window.getSelection && window.getSelection();
    if (!selection || !document.createRange) return;
    var range = document.createRange();
    range.selectNodeContents(node);
    selection.removeAllRanges();
    selection.addRange(range);
  }

  function initCopy(button) {
    if (button.hasAttribute(READY)) return;
    var label = button.textContent || "Copy";
    function say(text) {
      button.textContent = text;
      window.setTimeout(function () { button.textContent = label; }, 2500);
    }
    button.addEventListener("click", function () {
      var code = button.parentNode && button.parentNode.querySelector("code");
      if (!code) return;
      var text = code.textContent || "";
      function fallback() {
        selectText(code);
        var copied = false;
        try { copied = document.execCommand && document.execCommand("copy"); } catch (e) { copied = false; }
        say(copied ? "Copied" : "Selected: hold to copy");
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function () { say("Copied"); }, fallback);
      } else {
        fallback();
      }
    });
    button.setAttribute(READY, "");
  }

  function init(scope) {
    var root = scope || document;
    var nodes = root.querySelectorAll("[data-hub-compare]");
    for (var i = 0; i < nodes.length; i++) initCompare(nodes[i]);
    var groups = root.querySelectorAll("[data-hub-groups]");
    for (var j = 0; j < groups.length; j++) initGroups(groups[j]);
    var copies = root.querySelectorAll("[data-hub-copy]");
    for (var k = 0; k < copies.length; k++) initCopy(copies[k]);
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
