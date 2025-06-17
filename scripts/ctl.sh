####################################################################################################
# Script vars
####################################################################################################

kubernetes_context="docker-desktop"

website_container_image="website:latest-local"

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

configmap_exists () {
    local ns=$1
    local name=$2
    if [ "$(kubectl -n "$ns" get configmap -o json | jq --arg name "$name" -e '.items[] | select(.metadata.name == $name) | .metadata.name')" ]; then
        echo "configmap exists called $name in $ns"
        return 0
    fi
    return 1
}

####################################################################################################
# Operations
####################################################################################################

execute_operation() {
    local operation=$1

    case "$operation" in
        "start_website_dev")
            info_message "starting dev tools"

            if ! is_pid_running "AIR_PID" ; then
                air &
                export AIR_PID=$!
                info_message "'air' started with PID: $AIR_PID"
            fi
            ;;

        "stop_website_dev")
            info_message "stopping dev tools"

            if is_pid_running "AIR_PID" ; then
                info_message "killing 'air' with PID: $AIR_PID"
                kill "$AIR_PID"
                unset AIR_PID
            fi
            ;;

        "restart_website_dev")
            execute_operation "stop_website_dev"
            execute_operation "start_website_dev"
            ;;

        "build_containers")
            if ! docker build -t "$website_container_image" .; then
                echo "Failed to build website container"
                return 1
            fi
            ;;
        *)
            echo "Invalid operation: $operation"
            ;;
    esac
}

####################################################################################################
# Operation Helpers
####################################################################################################

value_in_array() {
    local value=$1
    local array=("${@:2}")

    for item in "${array[@]}"; do
        if [[ "$item" == "$value" ]]; then
            return 0
        fi
    done

    return 1
}

verbs=()

add_verb() {
    local verb=$1

    if ! value_in_array "$verb" "${verbs[@]}"; then
        verbs+=("$verb")
    fi
}

operations=()

add_operation() {
    local operation=$1

    if ! value_in_array "$operation" "${operations[@]}"; then
        operations+=("$operation")
    fi
}

is_pid_running() {
    local varname="$1"
    local pid="${(P)varname}"

    if [ -z "$pid" ]; then
        return 1
    fi

    if ps -p "$pid" > /dev/null 2>&1; then
        return 0
    fi
}

####################################################################################################
# Operations
####################################################################################################

if is_pid_running "AIR_PID" 
; then
    add_verb "stop"
    add_operation "stop_website_dev"

    add_verb "restart"
    add_operation "restart_website_dev"
else
    add_verb "start"
    add_operation "start_website_dev"
fi

add_verb "lint"
add_operation "lint_website"

add_verb "build"
add_operation "build_website"

add_verb "build"
add_operation "build_containers"

####################################################################################################
# Runtime
####################################################################################################

verbs_string=$(printf '%s\n' "${verbs[@]}")
operations_string=$(printf '%s\n' "${operations[@]}")
verb=$(printf '%s\n' "$verbs_string" | fzf --prompt="Select verb: " --height 40% --layout reverse --border --header "Verbs" --preview "echo \"$operations_string\" | grep ^{}")

operation=$(echo "$operations_string" | grep "^$verb" | fzf --prompt="Select operation: " --height 40% --layout reverse --border --header "Operations")

execute_operation "$operation"
