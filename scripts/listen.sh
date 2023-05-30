#!/bin/bash
# This script tracks logs on a remote server, sending an alert to your phone
# when one of the error codes you specify appears in the logs. This is useful
# for catching errors in production
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"
source "${HERE}/keylessSsh.sh"

# The error codes you want to track
error_codes='(9999|0192|4200)'

# Read arguments
while getopts "hi:l:m:v:" opt; do
    case $opt in
    i)
        INTERVAL=$OPTARG
        ;;
    l)
        WILL_LOOP=$OPTARG
        ;;
    m)
        MAX_EMAILS=$OPTARG
        ;;
    v)
        VERSION=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-h HELP] [-i INTERVAL] [-l LOOP] [-v VERSION]"
        echo "  -h --help: Show this help message"
        echo "  -i --interval: The interval in seconds for fetching the logs, if running on a loop"
        echo "  -l --loop: Whether to run this script on a loop, or to exit after one run"
        echo "  -m --max-emails: The maximum number of emails that can be sent per day. Default is 5."
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

# Default maximum daily emails to 5
if [ -z "$MAX_EMAILS" ]; then
    MAX_EMAILS=5
fi
EMAILS_SENT=0
# Set timestamp for last time EMAILS_SENT was reset
LAST_RESET=$(date +%s)
# Store the latest timestamp found in errors sent to email.
# This is used to prevent duplicate emails
LATEST_TIMESTAMP=0 #TODO

# Load variables from .env file
if [ -f "${HERE}/../.env" ]; then
    source "${HERE}/../.env"
else
    error "Could not find .env file. Exiting..."
    exit 1
fi

# Function to send an email to notify of the issue
send_email_alert() {
    # If the last time MAX_EMAILS was reset was more than 24 hours ago, reset it
    if [ $(($(date +%s) - $LAST_RESET)) -gt 86400 ]; then
        LAST_RESET=$(date +%s)
        EMAILS_SENT=0
    fi
    # If the maximum number of emails has been sent, exit
    if [ "$EMAILS_SENT" -ge "$MAX_EMAILS" ]; then
        error "Maximum number of error emails already sent"
        return
    fi
    # Increment EMAILS_SENT
    EMAILS_SENT=$((EMAILS_SENT + 1))

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
    info "Sent email alert to $LETSENCRYPT_EMAIL"
    # Sleep for a few seconds to avoid spamming the email server
    sleep 5
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

# Set the remote server's log directory path
remote_log_dir="/root/Vrooli/data/logs"

# Set the local directory to save the logs
local_dir="${HERE}/../data/logs-remote-${SITE_IP}-${VERSION}"

# Set the interval in seconds for fetching the logs
if [ -z "$INTERVAL" ]; then
    INTERVAL=1800 # default to 30 minutes
fi

if [ -z "$WILL_LOOP" ]; then
    WILL_LOOP=true
fi

# Get the last line checked for each log file. This is stored in a file called `listen-sh-status.txt` in the
# same directory that the logs are saved to. We store this in a file so that the script can be stopped and
# restarted without losing the last line checked for each log file.
last_lines_checked_file="${local_dir}/listen-sh-status.txt"
declare -A last_lines_checked
if [ -f "$last_lines_checked_file" ]; then
    info "Loading last lines checked from file: ${last_lines_checked_file}"
    while read line; do
        log_file=$(echo $line | awk -F'=' '{print $1}')
        last_line_checked=$(echo $line | awk -F'=' '{print $2}')
        last_lines_checked[$log_file]=$last_line_checked
    done <$last_lines_checked_file
fi
info "Last lines checked: ${last_lines_checked[@]}"

while true; do
    # Fetch all log files from the remote directory that have been modified since the last fetch
    info "Fetching logs from remote server at ${remote_log_dir}..."
    rsync -avz -e "ssh -i ~/.ssh/id_rsa_${SITE_IP}" --update --exclude ".*" $remote_server:$remote_log_dir $local_dir

    # Loop through all updated log files in the local directory
    for log_file in $(find $local_dir -type f -newermt "$(date +%Y-%m-%d -d 'yesterday')"); do
        # Get the last line checked in the log file, if it exists
        last_line_checked=${last_lines_checked[$log_file]}
        if [ -z "$last_line_checked" ]; then
            last_line_checked=""
        fi
        # Get the current last line of the log file
        file_last_line=$(tail -n 1 $log_file)
        # If the last line checked is the same as the current last line, skip the log file
        if [ "$last_line_checked" = "$file_last_line" ]; then
            info "No new lines in log file: $log_file"
            continue
        fi
        # Read the log file line by line in reverse order
        info "Checking log file for tracked errors: $log_file"
        # TODO for morning: This isn't checking errors properly (probably). Also look at search property in models/statsApi.ts and models/statsSite.ts. Replicate that on other stats models.
        while read -r line || [[ -n $line ]]; do
            # Check if the line has already been checked
            if [ "$line" = "$last_line_checked" ]; then
                break
            fi
            # Check if one of the error codes is present in the line.
            # Errors are stored in JSON, with the code in the format "trace":"code".
            # We can use this knowledge to make sure we only match the error code
            if [[ "${line} " =~ "code\":\""${error_codes}"-" ]]; then
                # Send an email alert
                send_email_alert $line $log_file
            fi
        done < <(tac "$log_file")
        # Update the last line checked for the log file
        last_lines_checked[$log_file]=$file_last_line
    done
    success "Finished fetching logs from remote server."

    # Save the last lines checked to the status file
    info "Saving last lines checked to file: ${last_lines_checked_file}"
    echo "" >$last_lines_checked_file
    for log_file in "${!last_lines_checked[@]}"; do
        echo "$log_file=${last_lines_checked[$log_file]}" >>$last_lines_checked_file
    done
    info "Finished saving last lines checked to file."

    # If not running on a loop, exit the script
    if [ "$WILL_LOOP" = false ]; then
        exit 0
    fi
    # Otherwise, wait for the specified interval before fetching the logs again
    info "Waiting $INTERVAL seconds before fetching logs again..."
    sleep $INTERVAL
done
