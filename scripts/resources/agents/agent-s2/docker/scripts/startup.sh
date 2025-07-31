#!/bin/bash
# Startup script for Agent S2 container

# Ensure runtime directory exists
mkdir -p /tmp/runtime-agents2
chmod 700 /tmp/runtime-agents2

# Set environment variables
export DISPLAY=:99
export XDG_RUNTIME_DIR=/tmp/runtime-agents2

# Create .Xauthority file if it doesn't exist
touch /home/agents2/.Xauthority
chmod 600 /home/agents2/.Xauthority

# Wait for X server to be ready
echo "Waiting for X server..."
for i in {1..30}; do
    if xdpyinfo -display :99 >/dev/null 2>&1; then
        echo "X server is ready"
        break
    fi
    sleep 1
done

# Configure Firefox preferences for automation
echo "Configuring Firefox for automation..."
mkdir -p /home/agents2/.mozilla/firefox/agent-s2
cat > /home/agents2/.mozilla/firefox/profiles.ini << EOF
[General]
StartWithLastProfile=1

[Profile0]
Name=agent-s2
IsRelative=1
Path=agent-s2
Default=1
EOF

# Create Firefox preferences for automation
cat > /home/agents2/.mozilla/firefox/agent-s2/user.js << 'EOF'
// Disable notifications
user_pref("dom.webnotifications.enabled", false);
user_pref("permissions.default.desktop-notification", 2);

// Disable popups
user_pref("dom.disable_open_during_load", true);
user_pref("privacy.popups.showBrowserMessage", false);

// Disable first-run pages
user_pref("browser.startup.firstrunSkipsHomepage", true);
user_pref("browser.startup.homepage_override.mstone", "ignore");
user_pref("datareporting.policy.firstRunURL", "");
user_pref("browser.startup.page", 0);

// Disable telemetry and updates
user_pref("datareporting.healthreport.uploadEnabled", false);
user_pref("app.update.enabled", false);
user_pref("app.update.auto", false);
user_pref("browser.newtabpage.activity-stream.feeds.telemetry", false);
user_pref("browser.newtabpage.activity-stream.telemetry", false);
user_pref("browser.ping-centre.telemetry", false);
user_pref("toolkit.telemetry.enabled", false);
user_pref("toolkit.telemetry.unified", false);
user_pref("toolkit.telemetry.server", "");

// Disable crash reporter
user_pref("browser.crashReports.unsubmittedCheck.enabled", false);
user_pref("browser.crashReports.unsubmittedCheck.autoSubmit2", false);

// Performance optimizations
user_pref("browser.cache.disk.enable", false);
user_pref("browser.cache.memory.enable", true);
user_pref("browser.cache.memory.capacity", 524288);

// Memory and stability optimizations
user_pref("browser.sessionhistory.max_entries", 10);
user_pref("browser.sessionhistory.max_total_viewers", 0);
user_pref("browser.sessionstore.interval", 300000);
user_pref("browser.sessionstore.max_tabs_undo", 5);
user_pref("browser.sessionstore.max_windows_undo", 2);
user_pref("browser.tabs.max_tabs_undo", 10);
user_pref("browser.tabs.unloadOnLowMemory", true);
user_pref("browser.tabs.min_inactive_duration_before_unload", 300000);
user_pref("javascript.options.mem.max", 512000);
user_pref("javascript.options.mem.gc_incremental_slice_ms", 10);
user_pref("dom.ipc.processCount", 4);
user_pref("dom.ipc.processCount.webIsolated", 1);
user_pref("layers.acceleration.disabled", false);
user_pref("gfx.webrender.all", false);
user_pref("gfx.webrender.enabled", false);

// Disable password manager prompts
user_pref("signon.rememberSignons", false);
user_pref("signon.autofillForms", false);

// Disable safe browsing (for automation)
user_pref("browser.safebrowsing.malware.enabled", false);
user_pref("browser.safebrowsing.phishing.enabled", false);

// Set download behavior
user_pref("browser.download.folderList", 2);
user_pref("browser.download.dir", "/home/agents2/Downloads");
user_pref("browser.download.useDownloadDir", true);
user_pref("browser.helperApps.neverAsk.saveToDisk", "application/pdf,application/octet-stream");

// Disable animations (faster automation)
user_pref("toolkit.cosmeticAnimations.enabled", false);

// ========== STEALTH MODE ENHANCEMENTS ==========

// Disable WebDriver detection
user_pref("dom.webdriver.enabled", false);
user_pref("useAutomationExtension", false);

// Disable automation indicators
user_pref("browser.newtabpage.activity-stream.improvesearch.handoffToAwesomebar", false);
user_pref("services.sync.prefs.sync.browser.newtabpage.activity-stream.showSponsoredTopSites", false);

// Enhanced privacy settings
user_pref("privacy.trackingprotection.enabled", true);
user_pref("privacy.trackingprotection.socialtracking.enabled", true);
user_pref("privacy.trackingprotection.cryptomining.enabled", true);
user_pref("privacy.trackingprotection.fingerprinting.enabled", true);

// Disable WebRTC leak
user_pref("media.peerconnection.enabled", false);
user_pref("media.navigator.enabled", false);
user_pref("media.navigator.video.enabled", false);
user_pref("media.getusermedia.audiocapture.enabled", false);
user_pref("media.getusermedia.screensharing.enabled", false);

// Disable battery API
user_pref("dom.battery.enabled", false);

// Disable gamepad API
user_pref("dom.gamepad.enabled", false);

// Disable sensors
user_pref("device.sensors.enabled", false);
user_pref("device.sensors.ambientLight.enabled", false);
user_pref("device.sensors.proximity.enabled", false);

// Enhanced fingerprinting resistance
user_pref("privacy.resistFingerprinting", true);
user_pref("privacy.resistFingerprinting.block_mozAddonManager", true);
user_pref("privacy.resistFingerprinting.letterboxing", false);

// Disable beacon
user_pref("beacon.enabled", false);

// Disable clipboard events
user_pref("dom.event.clipboardevents.enabled", false);

// Network settings
user_pref("network.http.referer.XOriginPolicy", 2);
user_pref("network.http.referer.XOriginTrimmingPolicy", 2);

// Disable prefetching
user_pref("network.prefetch-next", false);
user_pref("network.dns.disablePrefetch", true);
user_pref("network.predictor.enabled", false);

// Disable link prefetching
user_pref("network.http.speculative-parallel-limit", 0);

// Set timezone to UTC (fingerprinting defense)
user_pref("javascript.use_us_english_locale", true);

// Disable geolocation
user_pref("geo.enabled", false);
user_pref("geo.wifi.url", "");
user_pref("browser.search.geoip.url", "");

// Canvas fingerprinting protection (handled by resistFingerprinting)
user_pref("canvas.poisondata", true);

// WebGL protection
user_pref("webgl.disabled", false);
user_pref("webgl.min_capability_mode", true);
user_pref("webgl.disable-extensions", true);
user_pref("webgl.disable-fail-if-major-performance-caveat", true);
user_pref("webgl.enable-debug-renderer-info", false);

// Audio fingerprinting protection
user_pref("dom.audiocontext.enabled", true);
user_pref("media.webaudio.enabled", true);

// Font fingerprinting protection
user_pref("browser.display.use_document_fonts", 1);
user_pref("layout.css.font-visibility.level", 1);

// Hardware fingerprinting protection
user_pref("dom.maxHardwareConcurrency", 2);

// Disable resource timing API
user_pref("dom.enable_resource_timing", false);

// Disable timing attacks
user_pref("dom.enable_performance", false);
user_pref("dom.enable_performance_navigation_timing", false);

// Disable plugin scanning
user_pref("plugin.scan.plid.all", false);

// Set generic user agent (will be overridden by stealth profile)
user_pref("general.useragent.override", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36");

// Disable media devices enumeration
user_pref("media.navigator.permission.disabled", true);

// Disable speech synthesis
user_pref("media.webspeech.synth.enabled", false);

// Disable VR
user_pref("dom.vr.enabled", false);

// Disable vibration API
user_pref("dom.vibrator.enabled", false);

// Set DNT header
user_pref("privacy.donottrackheader.enabled", true);
user_pref("privacy.donottrackheader.value", 1);

// Session restoration (for persistence)
user_pref("browser.sessionstore.resume_from_crash", true);
user_pref("browser.sessionstore.restore_on_demand", true);
user_pref("browser.sessionstore.restore_tabs_lazily", true);

// Cookie settings (for session persistence)
user_pref("network.cookie.lifetimePolicy", 0);
user_pref("network.cookie.thirdparty.sessionOnly", false);

// Disable Firefox's automation detection
user_pref("marionette.enabled", false);
user_pref("dom.webdriver.enabled", false);

// ========== AUTOMATED SESSION RESET SUPPORT ==========

// Disable quit confirmation dialogs (for automated session clearing)
user_pref("browser.warnOnQuit", false);
user_pref("browser.showQuitWarning", false);
user_pref("browser.sessionstore.warnOnQuit", false);

// Disable tab close warnings (for automated tab management)  
user_pref("browser.tabs.warnOnClose", false);
user_pref("browser.tabs.warnOnCloseOtherTabs", false);

// Disable window close warnings
user_pref("browser.warnOnCloseWindow", false);
user_pref("browser.tabs.closeWindowWithLastTab", true);
EOF

# Ensure proper permissions
chown -R agents2:agents2 /home/agents2/.mozilla

# Note: Security proxy setup (iptables) is now handled by init.sh during container startup
# The mitmproxy service itself is started by the Python application

# Firefox is now managed by the firefox-monitor supervisor process
# This prevents crashes and provides automatic recovery
echo "Firefox will be managed by supervisor process..."

# Start xterm terminal (for emergency access)
echo "Starting terminal..."
DISPLAY=:99 xterm -geometry 80x24+10+10 -bg black -fg white -fa 'Monospace' -fs 10 > /var/log/supervisor/xterm.log 2>&1 &

# Set up Fluxbox background to prevent fbsetbg errors
echo "Setting up wallpaper..."
mkdir -p /home/agents2/.fluxbox
cat > /home/agents2/.fluxbox/overlay << 'EOF'
# Solid color background - prevents fbsetbg errors
background: solid
background.color: #2c3e50
EOF

# Create fbsetbg configuration to use feh as wallpaper setter
cat > /home/agents2/.fluxbox/lastwallpaper << 'EOF'
$full feh --bg-fill --no-fehbg '' 2>/dev/null || true
EOF
chmod +x /home/agents2/.fluxbox/lastwallpaper

# Create a simple desktop file for easy access
mkdir -p /home/agents2/Desktop
cat > /home/agents2/Desktop/firefox.desktop << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Firefox
Comment=Web Browser
Exec=firefox-esr
Icon=firefox
Terminal=false
Categories=Network;WebBrowser;
EOF

cat > /home/agents2/Desktop/terminal.desktop << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Terminal
Comment=Terminal Emulator
Exec=xterm
Icon=utilities-terminal
Terminal=false
Categories=System;TerminalEmulator;
EOF

chmod +x /home/agents2/Desktop/*.desktop
chown -R agents2:agents2 /home/agents2/Desktop

echo "Agent S2 container initialized successfully with Firefox and Terminal"