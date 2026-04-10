#!/usr/bin/env bash

vnc::export_config() {
    export VNC_NAME="vnc"
    export VNC_DISPLAY_NAME="VNC Server"
    export VNC_DESCRIPTION="Virtual Network Computing server for remote display access via x11vnc and websockify"
    export VNC_DATA_DIR="${HOME}/.vrooli/vnc"
    export VNC_SESSIONS_DIR="${VNC_DATA_DIR}/sessions"
    export VNC_LOGS_DIR="${VNC_DATA_DIR}/logs"
    export VNC_PORT_START=5900
    export VNC_PORT_END=5999
    export VNC_WS_PORT_START=6080
    export VNC_WS_PORT_END=6180
}

vnc::export_config
