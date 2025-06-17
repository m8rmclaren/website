#!/usr/bin/env bash

####################################################################################################
# Script vars
####################################################################################################

env_file=".env"
ctl_script="scripts/ctl.sh"
aliasname="dev"

BOLD="\033[1m"
RESET="\033[0m"
CYAN="\033[36m"
ORANGE="\033[0;33m"
RED="\033[31m"

####################################################################################################
# Helpers
####################################################################################################

print_divider () {
    local message="$1"
    echo -e "\n${BOLD}==========  ${message}  ========== ${RESET}\n"
}

info_message () {
    local message="$1"
    local mtype="INFO"
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo -e "${CYAN}${timestamp} [$mtype] ${RESET}$message"
}

warning_message () {
    local message="$1"
    local mtype="WARNING"
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo -e "${ORANGE}${timestamp} [$mtype] ${RESET}$message"
}

error_message () {
    local message="$1"
    local mtype="YIKES"
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo -e "${RED}${timestamp} [$mtype] $1${RESET}"
}

####################################################################################################
# Generate .env file for local dev
####################################################################################################

info_message "Generating .env for local dev"

# Overwrite any existing .env; remove '>' below if you want to append instead
cat > "$env_file" <<EOF
eval "function $aliasname() { source $(pwd)/$ctl_script; }"
alias "$aliasname=$aliasname"
EOF

print_divider "Done!"

info_message "Please run 'source .env'"


