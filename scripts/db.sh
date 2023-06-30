#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

info 'Getting secrets...'
${HERE}/getSecrets.sh -e ${NODE_ENV} -s DB_PASSWORD

# start the database
exec docker-entrypoint.sh postgres
