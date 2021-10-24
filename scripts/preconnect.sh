#!/bin/sh
# Updates index.html file to best match the current environment

# Replace PRECONNECT variable in index.html
Preconnect="http://localhost:5000"
if [ "${REACT_APP_SERVER_LOCATION}" = "dns" ]; then
    Preconnect="https://${REACT_APP_SITE_NAME}"
fi
sed -i "s|<PRECONNECT>|${Preconnect}|g" "${PROJECT_DIR}/packages/ui/build/index.html"

# while read line; do
#   value=${line#*=} | tr -d '"'
#   name=${line%%=*}
#   echo "V: $value"
#   echo "N: $name"
# done <.env