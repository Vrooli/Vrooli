export function showSnackbar(message, type = "info", duration = 3000) {
  const snackbar = document.createElement("div");
  snackbar.className = `snackbar snackbar-${type}`;
  snackbar.textContent = message;
  snackbar.style.cssText = `
    position: fixed;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%) translateY(100px);
    padding: 12px 24px;
    background: ${type === "success" ? "#10b981" : type === "error" ? "#ef4444" : "#3b82f6"};
    color: white;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    z-index: 320000;
    font-size: 14px;
    font-weight: 500;
    opacity: 0;
    pointer-events: none;
    transition: transform 0.3s ease, opacity 0.3s ease;
  `;

  document.body.appendChild(snackbar);

  requestAnimationFrame(() => {
    snackbar.style.transform = "translateX(-50%) translateY(0)";
    snackbar.style.opacity = "1";
  });

  setTimeout(() => {
    snackbar.style.transform = "translateX(-50%) translateY(100px)";
    snackbar.style.opacity = "0";
    setTimeout(() => snackbar.remove(), 300);
  }, duration);
}
