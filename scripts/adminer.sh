#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

info 'Getting secrets...'
. ${HERE}/getSecrets.sh -e ${NODE_ENV} -s DB_PASSWORD

# start adminer
exec php -S [::]:8080 -t /var/www/html/
