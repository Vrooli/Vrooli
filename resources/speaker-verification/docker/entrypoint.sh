#!/usr/bin/env bash
set -e

echo "=== Speaker Verification Service ==="
echo "Device: ${SPEAKER_VERIFICATION_DEVICE:-auto}"
echo "Model:  ${SPEAKER_VERIFICATION_MODEL:-nvidia/speakerverification_en_titanet_large}"
echo

# Ensure data directories exist
mkdir -p /data/profiles /data/cache /data/logs

exec uvicorn app:app \
    --host 0.0.0.0 \
    --port 8891 \
    --log-level info \
    --no-access-log
