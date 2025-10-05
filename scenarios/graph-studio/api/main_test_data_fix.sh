#!/bin/bash
# Fix the Data field issue by converting to json.RawMessage
sed -i 's/Data: map\[string\]interface{}/Data: json.RawMessage(`/g' main_test.go
# Note: This is a simplified fix - we need to properly fix each occurrence
