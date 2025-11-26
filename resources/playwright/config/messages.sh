msg_resource_starting() {
  echo "Starting Playwright driver on ${PLAYWRIGHT_DRIVER_HOST:-127.0.0.1}:${PLAYWRIGHT_DRIVER_PORT:-39400}..."
}

msg_resource_started() {
  echo "Playwright driver started (pid $(cat /tmp/vrooli-playwright-driver.pid 2>/dev/null || echo '?'))."
}

msg_resource_stopping() {
  echo "Stopping Playwright driver..."
}

msg_resource_stopped() {
  echo "Playwright driver stopped."
}
