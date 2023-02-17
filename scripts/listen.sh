#!/bin/bash
# This script tracks logs on a remote server, sending an alert to your phone
# when one of the error codes you specify appears in the logs. This is useful
# for catching errors in production
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"
source "${HERE}/keylessSsh.sh"

# The error codes you want to track
error_codes=("0000" "0001" "0002")

# Read arguments
while getopts ":h:i:l:v" opt; do
    case $opt in
    i)
        INTERVAL=$OPTARG
        ;;
    l)
        WILL_LOOP=$OPTARG
        ;;
    v)
        VERSION=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-h HELP] [-i INTERVAL] [-l LOOP] [-v VERSION]"
        echo "  -h --help: Show this help message"
        echo "  -i --interval: The interval in seconds for fetching the logs, if running on a loop"
        echo "  -l --loop: Whether to run this script on a loop, or to exit after one run"
        echo "  -v --version: Version number of site on server (e.g. \"1.0.0\"). Used to locate the database directory"
        exit 0
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done

# Load variables from .env file
if [ -f "${HERE}/../.env" ]; then
    source "${HERE}/../.env"
else
    error "Could not find .env file. Exiting..."
    exit 1
fi

# Function to send an email to notify of the issue
send_email_alert() {
    # Set the email subject
    subject="[ERROR ALERT] Log Monitoring Script"

    # Parse the error line into separate fields
    error_line="$1"
    log_file="$2"
    error_code=$(echo $error_line | grep -o '"trace":"[^"]*"' | awk -F '"' '{print $4}')
    error_message=$(echo $error_line | grep -o '"msg":"[^"]*"' | awk -F '"' '{print $4}')
    timestamp=$(echo $error_line | grep -o '"timestamp":"[^"]*"' | awk -F '"' '{print $4}')

    # Set the email body
    body="An error was detected in the log file: $log_file
Error code: $error_code
Error message: $error_message
Timestamp: $timestamp

Full error line:
$error_line"

    # Send the email using the 'mail' command
    echo "$body" | mail -s "$subject" "$LETSENCRYPT_EMAIL"
}

# Set the remote server location, using SITE_IP from .env
remote_server="root@${SITE_IP}"
info "Remote server: ${remote_server}"

# If no version number is defined, use the version number found in the package.json files
if [ -z "$VERSION" ]; then
    info "No version number defined. Using version number found in package.json files."
    VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
    info "Version number found in package.json files: ${VERSION}"
fi

# Set the remote server's log directory path using the version number
remote_log_dir="${SITE_IP}:/var/tmp/${VERSION}/data/logs"

# Set the local directory to save the logs
local_dir="${HERE}/../data/logs-remote-${SITE_IP}-${VERSION}"

# Set the interval in seconds for fetching the logs
if [ -z "$INTERVAL" ]; then
    INTERVAL=1800 # default to 30 minutes
fi

while true; do
    # Fetch all log files from the remote directory that have been modified since the last fetch
    rsync -avz --update --exclude ".*" $remote_server:$remote_log_dir $local_dir

    # Loop through all updated log files in the local directory
    for log_file in $(find $local_dir -type f -newermt "$(date +%Y-%m-%d -d 'yesterday')"); do
        # Read the log file line by line in reverse order
        while read -r line || [[ -n $line ]]; do
            # Check if one of the error codes is present in the line.
            # Errors are stored in JSON, with the code in the format "trace":"code".
            # We can use this knowledge to make sure we only match the error code
            if [[ " ${line} " =~ "trace\":\"[0-9]{4}" ]]; then
                # Send an email alert
                send_email_alert $line $log_file
            fi
        done < <(tac "$log_file")
    done

    # If not running on a loop, exit the script
    if [ "$WILL_LOOP" = false ]; then
        exit 0
    fi
    # Otherwise, wait for the specified interval before fetching the logs again
    sleep $INTERVAL
done
