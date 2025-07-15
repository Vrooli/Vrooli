#!/bin/bash
# Simple beep script for notification purposes
# This script plays a beep sound using PowerShell

# Play a single subtle "tick" sound - low but audible frequency, extremely short duration
powershell.exe -c "[console]::beep(500, 5)"