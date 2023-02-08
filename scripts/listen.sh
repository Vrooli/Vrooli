#!/bin/bash
# This script tracks logs on a remote server, sending an alert to your phone
# when one of the error codes you specify appears in the logs. This is useful 
# for catching errors in production
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# The error codes you want to track
error_codes=("0000" "0001" "0002")

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

# Ask for current deployed version number, which is needed to locate the logs
if [ -z "$VERSION" ]; then
    prompt "What version is running on the server? (e.g. 1.0.0). Leave this blank to auto-detect using local version."
    read -r VERSION
    # If no version number was entered, use the version number found in the package.json files
    if [ -z "$VERSION" ]; then
        info "No version number entered. Using version number found in package.json files."
        VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
        info "Version number found in package.json files: ${VERSION}"
    fi
fi

# Set the remote server's log directory path using the version number
remote_log_dir="${SITE_IP}:/var/tmp/${VERSION}/data/logs"

# Set the local directory to save the logs
local_dir="${HERE}/../data/logs-remote-${SITE_IP}-${VERSION}"

# Set the interval in seconds for fetching the logs
interval=1800

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

  # Wait for the specified interval before fetching the logs again
  sleep $interval
done