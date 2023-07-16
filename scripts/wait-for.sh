#!/bin/sh
set -e

usage() {
    echo "Usage: $0 HOST:PORT [-t timeout] [-- command]"
    exit 1
}

HOST=
PORT=
TIMEOUT=15
CMD=

# Process arguments
for i in "$@"; do
    case $i in
    *:*)
        HOST=$(echo $1 | cut -d : -f 1)
        PORT=$(echo $1 | cut -d : -f 2)
        shift
        ;;
    -t)
        TIMEOUT="$2"
        shift 2
        ;;
    --)
        shift
        CMD="${@}"
        break
        ;;
    *) ;;
    esac
done

if [ -z "$HOST" ] || [ -z "$PORT" ]; then
    usage
fi

# Wait for the host:port to be available
for i in $(seq $TIMEOUT); do
    if nc -z "$HOST" "$PORT" >/dev/null 2>&1; then
        if [ -n "$CMD" ]; then
            exec $CMD
        fi
        exit 0
    fi
    sleep 1
done

echo >&2 "Operation timed out"
exit 0 # Don't fail on timeout
