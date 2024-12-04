#!/bin/sh
#
# This script is used to compile your program on CodeCrafters
# 
# This runs before .codecrafters/run.sh
#
# Learn more: https://codecrafters.io/program-interface

# Exit early if any commands fail
set -e

sed -i 's/debugger{enabled: true}/debugger{enabled: false}/g' cmd/myshell/main.go

go build -o /tmp/shell-target cmd/myshell/*.go
