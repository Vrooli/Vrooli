# === BUILD STAGE ===
FROM python:3.11-slim as builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc python3-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python packages
COPY docker/images/agent-s2/requirements.txt /tmp/
RUN pip install --user --no-cache-dir -r /tmp/requirements.txt && \
    pip install --user --no-cache-dir mitmproxy==10.1.6

# === RUNTIME STAGE ===
FROM python:3.11-slim as runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    # X11 and display server
    xvfb x11vnc fluxbox xauth \
    # GUI toolkit support
    python3-tk python3-dev \
    # Browser automation support
    libnss3 libgconf-2-4 libxss1 libgdk-pixbuf2.0-0 \
    libgtk-3-0 libxcomposite1 libxdamage1 libxrandr2 \
    libappindicator3-1 xdg-utils \
    # Fonts for proper rendering
    fonts-liberation fonts-dejavu-core fonts-noto-color-emoji \
    # Process management
    supervisor \
    # Networking and utilities
    curl net-tools procps \
    # Security proxy tools
    sudo iptables iproute2 libnss3-tools \
    # Required for pyautogui
    python3-xlib scrot gnome-screenshot \
    # Demo GUI applications for testing and automation examples
    xterm mousepad gedit firefox-esr gnome-calculator \
    # Image viewer and wallpaper setter (fixes fbsetbg error)
    feh \
    # Window management tools for target-aware automation
    wmctrl xdotool x11-utils \
    # Clean up
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user with proper groups
RUN groupadd -g 1000 agents2 \
    && useradd -m -u 1000 -g agents2 -G audio,video agents2 \
    && mkdir -p /home/agents2/.agent-s2 /home/agents2/.cache \
    && chown -R agents2:agents2 /home/agents2

# Copy Python packages from builder
COPY --from=builder /root/.local /home/agents2/.local

# Set up Python path
ENV PYTHONPATH="/home/agents2/.local/lib/python3.11/site-packages:${PYTHONPATH}"
ENV PATH="/home/agents2/.local/bin:${PATH}"

# Create necessary directories
RUN mkdir -p /opt/agent-s2/api /var/log/supervisor /tmp/.X11-unix \
    && chmod 1777 /tmp/.X11-unix \
    && chown agents2:agents2 /opt/agent-s2 /var/log/supervisor

# Copy configuration files
COPY --chown=agents2:agents2 docker/config/supervisor.conf /etc/supervisor/conf.d/

# Copy scripts (init.sh must be owned by root for privileged operations)
COPY docker/scripts/init.sh /opt/agent-s2/
COPY --chown=agents2:agents2 docker/scripts/startup.sh /opt/agent-s2/
COPY --chown=agents2:agents2 docker/scripts/vnc-password.sh /opt/agent-s2/
COPY --chown=agents2:agents2 docker/scripts/setup-proxy.sh /opt/agent-s2/
COPY --chown=agents2:agents2 docker/scripts/firefox-monitor.sh /opt/agent-s2/

# Copy Python package
COPY --chown=agents2:agents2 agent_s2 /opt/agent-s2/agent_s2/

# Make scripts executable
RUN chmod +x /opt/agent-s2/*.sh

# Set up VNC password
ARG VNC_PASSWORD=agents2vnc
RUN /opt/agent-s2/vnc-password.sh "$VNC_PASSWORD"

# Make init script executable and set up sudo permissions for iptables
RUN chmod +x /opt/agent-s2/init.sh && \
    echo "agents2 ALL=(ALL) NOPASSWD: /sbin/iptables, /sbin/ip6tables, /usr/sbin/sysctl" >> /etc/sudoers.d/agents2-proxy && \
    chmod 0440 /etc/sudoers.d/agents2-proxy

# Switch to non-root user (init script will use sudo for iptables)
USER agents2
WORKDIR /home/agents2

# Environment variables
ENV DISPLAY=:99
ENV AGENT_S2_USER=agents2
ENV HOME=/home/agents2
ENV XDG_RUNTIME_DIR=/tmp/runtime-agents2
ENV PYTHONPATH=/opt/agent-s2:$PYTHONPATH
# Firefox crash prevention
ENV MOZ_CRASHREPORTER_DISABLE=1
ENV MOZ_DISABLE_CONTENT_SANDBOX=1
ENV MOZ_FORCE_DISABLE_E10S=1

# AI Configuration environment variables
ENV AGENT_S2_AI_PROVIDER=ollama
ENV AGENT_S2_AI_MODEL=llama3.2-vision:11b
ENV AGENT_S2_AI_ENABLED=true
ENV AGENT_S2_SEARCH_ENABLED=false
ENV AGENT_S2_AI_TIMEOUT=300

# Create runtime directory
RUN mkdir -p /tmp/runtime-agents2 && chmod 700 /tmp/runtime-agents2

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:4113/health || exit 1

# Expose ports
EXPOSE 4113 5900

# Start with init script (runs as root for privileged setup, then starts supervisor)
CMD ["/opt/agent-s2/init.sh"]